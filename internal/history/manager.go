package history

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Manager は調査履歴を管理する
type Manager struct {
	mu         sync.RWMutex
	filePath   string
	maxEntries int
	history    *History
}

// NewManager は新しいManagerを作成する
func NewManager(filePath string, maxEntries int) *Manager {
	return &Manager{
		filePath:   filePath,
		maxEntries: maxEntries,
		history:    nil,
	}
}

// Load はファイルから履歴を読み込む
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// ファイルが存在しない場合は新規作成
			m.history = &History{
				Version:     1,
				LastUpdated: time.Now(),
				Entries:     []Entry{},
				Index: Index{
					ByKeyword: make(map[string][]string),
					ByType:    make(map[string][]string),
					ByDate:    make(map[string][]string),
				},
			}
			return nil
		}
		return fmt.Errorf("failed to read research history file: %w", err)
	}

	var h History
	if err := json.Unmarshal(data, &h); err != nil {
		return fmt.Errorf("failed to unmarshal research history: %w", err)
	}

	m.history = &h
	return nil
}

// Save は履歴をファイルに保存する
func (m *Manager) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.history == nil {
		return nil
	}

	m.history.LastUpdated = time.Now()

	data, err := json.MarshalIndent(m.history, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal research history: %w", err)
	}

	if err := os.WriteFile(m.filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write research history file: %w", err)
	}

	return nil
}

// AddEntry は新しいエントリを追加する
func (m *Manager) AddEntry(entry Entry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.history == nil {
		m.history = &History{
			Version:     1,
			LastUpdated: time.Now(),
			Entries:     []Entry{},
			Index: Index{
				ByKeyword: make(map[string][]string),
				ByType:    make(map[string][]string),
				ByDate:    make(map[string][]string),
			},
		}
	}

	// IDを生成
	if entry.ID == "" {
		entry.ID = "req_" + uuid.New().String()[:8]
	}

	// タイムスタンプを設定
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	// キーワードを抽出
	if len(entry.Keywords) == 0 {
		entry.Keywords = extractKeywords(entry.Query)
	}

	// エントリを追加
	m.history.Entries = append(m.history.Entries, entry)

	// インデックスを更新
	m.updateIndex(entry)

	// 古いエントリを削除
	m.pruneUnlocked()

	return nil
}

// updateIndex はインデックスを更新する（ロック済みで呼び出されること）
func (m *Manager) updateIndex(entry Entry) {
	// by_keyword インデックス
	for _, keyword := range entry.Keywords {
		kw := strings.ToLower(keyword)
		if m.history.Index.ByKeyword == nil {
			m.history.Index.ByKeyword = make(map[string][]string)
		}
		m.history.Index.ByKeyword[kw] = append(m.history.Index.ByKeyword[kw], entry.ID)
	}

	// by_type インデックス
	if m.history.Index.ByType == nil {
		m.history.Index.ByType = make(map[string][]string)
	}
	m.history.Index.ByType[entry.Type] = append(m.history.Index.ByType[entry.Type], entry.ID)

	// by_date インデックス
	dateKey := entry.Timestamp.Format("2006-01-02")
	if m.history.Index.ByDate == nil {
		m.history.Index.ByDate = make(map[string][]string)
	}
	m.history.Index.ByDate[dateKey] = append(m.history.Index.ByDate[dateKey], entry.ID)
}

// pruneUnlocked は古いエントリを削除する（ロック済みで呼び出されること）
func (m *Manager) pruneUnlocked() {
	if len(m.history.Entries) <= m.maxEntries {
		return
	}

	// 削除するエントリのIDを収集
	deleteCount := len(m.history.Entries) - m.maxEntries
	deletedIDs := make(map[string]bool)
	for i := 0; i < deleteCount; i++ {
		deletedIDs[m.history.Entries[i].ID] = true
	}

	// エントリを削除
	m.history.Entries = m.history.Entries[deleteCount:]

	// インデックスから削除されたIDを除去
	m.cleanupIndex(deletedIDs)
}

// cleanupIndex はインデックスから削除されたIDを除去する
func (m *Manager) cleanupIndex(deletedIDs map[string]bool) {
	// by_keyword
	for keyword, ids := range m.history.Index.ByKeyword {
		filtered := filterIDs(ids, deletedIDs)
		if len(filtered) == 0 {
			delete(m.history.Index.ByKeyword, keyword)
		} else {
			m.history.Index.ByKeyword[keyword] = filtered
		}
	}

	// by_type
	for typ, ids := range m.history.Index.ByType {
		filtered := filterIDs(ids, deletedIDs)
		if len(filtered) == 0 {
			delete(m.history.Index.ByType, typ)
		} else {
			m.history.Index.ByType[typ] = filtered
		}
	}

	// by_date
	for date, ids := range m.history.Index.ByDate {
		filtered := filterIDs(ids, deletedIDs)
		if len(filtered) == 0 {
			delete(m.history.Index.ByDate, date)
		} else {
			m.history.Index.ByDate[date] = filtered
		}
	}
}

// filterIDs は削除対象のIDを除外する
func filterIDs(ids []string, deletedIDs map[string]bool) []string {
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		if !deletedIDs[id] {
			result = append(result, id)
		}
	}
	return result
}

// SearchByKeyword はキーワードで検索する
func (m *Manager) SearchByKeyword(keyword string) []Entry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.history == nil {
		return nil
	}

	kw := strings.ToLower(keyword)
	ids, ok := m.history.Index.ByKeyword[kw]
	if !ok {
		return nil
	}

	return m.getEntriesByIDs(ids)
}

// SearchByType はタイプで検索する
func (m *Manager) SearchByType(entryType string) []Entry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.history == nil {
		return nil
	}

	ids, ok := m.history.Index.ByType[entryType]
	if !ok {
		return nil
	}

	return m.getEntriesByIDs(ids)
}

// GetRecent は直近n件を取得する
func (m *Manager) GetRecent(n int) []Entry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.history == nil || len(m.history.Entries) == 0 {
		return nil
	}

	entries := m.history.Entries
	if len(entries) <= n {
		// コピーして返す
		result := make([]Entry, len(entries))
		copy(result, entries)
		// 新しい順にソート
		sort.Slice(result, func(i, j int) bool {
			return result[i].Timestamp.After(result[j].Timestamp)
		})
		return result
	}

	// 直近n件を取得
	result := make([]Entry, n)
	copy(result, entries[len(entries)-n:])
	// 新しい順にソート
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.After(result[j].Timestamp)
	})
	return result
}

// GetByID は特定のIDのエントリを取得する
func (m *Manager) GetByID(id string) *Entry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.history == nil {
		return nil
	}

	for i := range m.history.Entries {
		if m.history.Entries[i].ID == id {
			entry := m.history.Entries[i]
			return &entry
		}
	}
	return nil
}

// getEntriesByIDs はIDリストからエントリを取得する（ロック済みで呼び出されること）
func (m *Manager) getEntriesByIDs(ids []string) []Entry {
	idSet := make(map[string]bool)
	for _, id := range ids {
		idSet[id] = true
	}

	result := make([]Entry, 0, len(ids))
	for _, entry := range m.history.Entries {
		if idSet[entry.ID] {
			result = append(result, entry)
		}
	}

	// 新しい順にソート
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.After(result[j].Timestamp)
	})

	return result
}

// stopWords は英語・日本語のストップワード (キーワード抽出時に除外する語)
var stopWords = map[string]bool{
	// English
	"a": true, "an": true, "the": true, "and": true, "or": true,
	"is": true, "are": true, "was": true, "were": true, "be": true,
	"to": true, "of": true, "in": true, "for": true, "on": true,
	"with": true, "at": true, "by": true, "from": true, "as": true,
	"how": true, "what": true, "why": true, "when": true, "where": true,
	"which": true, "who": true, "that": true, "this": true, "it": true,
	"i": true, "you": true, "we": true, "they": true, "he": true, "she": true,
	"do": true, "does": true, "did": true, "can": true, "could": true,
	"will": true, "would": true, "should": true, "have": true, "has": true,
	"": true,
	// Japanese particles (助詞)
	"は": true, "が": true, "の": true, "に": true, "を": true,
	"で": true, "と": true, "も": true, "や": true, "へ": true,
	"か": true, "な": true, "ね": true, "よ": true,
	"から": true, "まで": true, "より": true, "ので": true, "のに": true,
	"けど": true, "だけ": true, "しか": true,
	// Japanese demonstratives / copula
	"これ": true, "それ": true, "あれ": true,
	"この": true, "その": true, "あの": true,
	"です": true, "ます": true,
}

// wordSplitter は Unicode 文字・数字以外で分割する正規表現
var wordSplitter = regexp.MustCompile(`[^\p{L}\p{N}]+`)

// extractKeywords はクエリからキーワードを抽出する (英語・日本語対応)
func extractKeywords(query string) []string {
	query = strings.ToLower(query)
	words := wordSplitter.Split(query, -1)

	seen := make(map[string]bool)
	keywords := make([]string, 0)
	for _, word := range words {
		// バイト長 < 2 で単一 ASCII 文字を除外 (CJK 1文字は 3バイト以上なので通過)
		if len(word) < 2 {
			continue
		}
		if stopWords[word] || seen[word] {
			continue
		}
		seen[word] = true
		keywords = append(keywords, word)
	}

	return keywords
}

// GenerateRequestID は新しいリクエストIDを生成する
func GenerateRequestID() string {
	return "req_" + uuid.New().String()[:8]
}
