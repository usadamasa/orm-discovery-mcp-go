package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ResearchHistory は調査履歴全体を保持する構造体
type ResearchHistory struct {
	Version     int                  `json:"version"`
	LastUpdated time.Time            `json:"last_updated"`
	Entries     []ResearchEntry      `json:"entries"`
	Index       ResearchHistoryIndex `json:"index"`
}

// ResearchHistoryIndex は検索用インデックス
type ResearchHistoryIndex struct {
	ByKeyword map[string][]string `json:"by_keyword"`
	ByType    map[string][]string `json:"by_type"`
	ByDate    map[string][]string `json:"by_date"`
}

// ResearchEntry は個々の調査エントリ
type ResearchEntry struct {
	ID            string                 `json:"id"`
	Timestamp     time.Time              `json:"timestamp"`
	Type          string                 `json:"type"` // "search" or "question"
	Query         string                 `json:"query"`
	Keywords      []string               `json:"keywords"`
	ToolName      string                 `json:"tool_name"`
	Parameters    map[string]interface{} `json:"parameters,omitempty"`
	ResultSummary ResultSummary          `json:"result_summary"`
	DurationMs    int64                  `json:"duration_ms"`
}

// ResultSummary は結果のサマリー（タイプ別に異なる構造）
type ResultSummary struct {
	// search 用
	Count      int                `json:"count,omitempty"`
	TopResults []TopResultSummary `json:"top_results,omitempty"`

	// question 用
	AnswerPreview string `json:"answer_preview,omitempty"`
	SourcesCount  int    `json:"sources_count,omitempty"`
	FollowupCount int    `json:"followup_count,omitempty"`
}

// TopResultSummary は検索結果のトップ結果サマリー
type TopResultSummary struct {
	Title     string `json:"title"`
	Author    string `json:"author,omitempty"`
	ProductID string `json:"product_id,omitempty"`
}

// ResearchHistoryManager は調査履歴を管理する
type ResearchHistoryManager struct {
	mu         sync.RWMutex
	filePath   string
	maxEntries int
	history    *ResearchHistory
}

// NewResearchHistoryManager は新しいResearchHistoryManagerを作成する
func NewResearchHistoryManager(filePath string, maxEntries int) *ResearchHistoryManager {
	return &ResearchHistoryManager{
		filePath:   filePath,
		maxEntries: maxEntries,
		history:    nil,
	}
}

// Load はファイルから履歴を読み込む
func (m *ResearchHistoryManager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// ファイルが存在しない場合は新規作成
			m.history = &ResearchHistory{
				Version:     1,
				LastUpdated: time.Now(),
				Entries:     []ResearchEntry{},
				Index: ResearchHistoryIndex{
					ByKeyword: make(map[string][]string),
					ByType:    make(map[string][]string),
					ByDate:    make(map[string][]string),
				},
			}
			return nil
		}
		return fmt.Errorf("failed to read research history file: %w", err)
	}

	var history ResearchHistory
	if err := json.Unmarshal(data, &history); err != nil {
		return fmt.Errorf("failed to unmarshal research history: %w", err)
	}

	m.history = &history
	return nil
}

// Save は履歴をファイルに保存する
func (m *ResearchHistoryManager) Save() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

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
func (m *ResearchHistoryManager) AddEntry(entry ResearchEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.history == nil {
		m.history = &ResearchHistory{
			Version:     1,
			LastUpdated: time.Now(),
			Entries:     []ResearchEntry{},
			Index: ResearchHistoryIndex{
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
func (m *ResearchHistoryManager) updateIndex(entry ResearchEntry) {
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
func (m *ResearchHistoryManager) pruneUnlocked() {
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
func (m *ResearchHistoryManager) cleanupIndex(deletedIDs map[string]bool) {
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
func (m *ResearchHistoryManager) SearchByKeyword(keyword string) []ResearchEntry {
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
func (m *ResearchHistoryManager) SearchByType(entryType string) []ResearchEntry {
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
func (m *ResearchHistoryManager) GetRecent(n int) []ResearchEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.history == nil || len(m.history.Entries) == 0 {
		return nil
	}

	entries := m.history.Entries
	if len(entries) <= n {
		// コピーして返す
		result := make([]ResearchEntry, len(entries))
		copy(result, entries)
		// 新しい順にソート
		sort.Slice(result, func(i, j int) bool {
			return result[i].Timestamp.After(result[j].Timestamp)
		})
		return result
	}

	// 直近n件を取得
	result := make([]ResearchEntry, n)
	copy(result, entries[len(entries)-n:])
	// 新しい順にソート
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.After(result[j].Timestamp)
	})
	return result
}

// GetByID は特定のIDのエントリを取得する
func (m *ResearchHistoryManager) GetByID(id string) *ResearchEntry {
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
func (m *ResearchHistoryManager) getEntriesByIDs(ids []string) []ResearchEntry {
	idSet := make(map[string]bool)
	for _, id := range ids {
		idSet[id] = true
	}

	result := make([]ResearchEntry, 0, len(ids))
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

// extractKeywords はクエリからキーワードを抽出する
func extractKeywords(query string) []string {
	// 小文字に変換
	query = strings.ToLower(query)

	// 単語以外の文字で分割
	re := regexp.MustCompile(`[^a-z0-9]+`)
	words := re.Split(query, -1)

	// ストップワードを定義
	stopWords := map[string]bool{
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
	}

	// 重複を除去しつつキーワードを収集
	seen := make(map[string]bool)
	keywords := make([]string, 0)
	for _, word := range words {
		if len(word) < 2 {
			continue
		}
		if stopWords[word] {
			continue
		}
		if seen[word] {
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
