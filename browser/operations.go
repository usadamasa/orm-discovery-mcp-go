package browser

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/chromedp/chromedp"
)

// GetCollectionsFromHomePage はホームページからコレクション一覧を取得します
func (bc *BrowserClient) GetCollectionsFromHomePage() ([]map[string]interface{}, error) {
	log.Printf("ホームページからコレクション一覧を取得します")
	
	var collections []map[string]interface{}
	
	err := chromedp.Run(bc.ctx,
		// ホームページに移動
		chromedp.Navigate("https://learning.oreilly.com/home/"),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		
		// コレクション要素を待機
		chromedp.Sleep(3*time.Second), // ページの読み込み完了を待機
		
		// コレクション情報を取得
		chromedp.ActionFunc(func(ctx context.Context) error {
			// コレクションのタイトルを取得
			var titles []string
			if err := chromedp.Evaluate(`
				Array.from(document.querySelectorAll('[data-testid*="collection"], .collection-card, .playlist-card')).map(el => {
					const titleEl = el.querySelector('h2, h3, .title, [data-testid*="title"]');
					return titleEl ? titleEl.textContent.trim() : '';
				}).filter(title => title !== '')
			`, &titles).Do(ctx); err == nil && len(titles) > 0 {
				for i, title := range titles {
					collections = append(collections, map[string]interface{}{
						"id":    fmt.Sprintf("collection_%d", i+1),
						"title": title,
						"type":  "collection",
						"source": "homepage",
					})
				}
			}
			
			// プレイリストのタイトルも取得
			var playlists []string
			if err := chromedp.Evaluate(`
				Array.from(document.querySelectorAll('.playlist, [data-testid*="playlist"]')).map(el => {
					const titleEl = el.querySelector('h2, h3, .title, [data-testid*="title"]');
					return titleEl ? titleEl.textContent.trim() : '';
				}).filter(title => title !== '')
			`, &playlists).Do(ctx); err == nil && len(playlists) > 0 {
				for i, title := range playlists {
					collections = append(collections, map[string]interface{}{
						"id":    fmt.Sprintf("playlist_%d", i+1),
						"title": title,
						"type":  "playlist",
						"source": "homepage",
					})
				}
			}
			
			return nil
		}),
	)

	if err != nil {
		return nil, fmt.Errorf("ホームページからのコレクション取得でエラーが発生しました: %w", err)
	}

	log.Printf("ホームページから%d個のコレクションを取得しました", len(collections))
	return collections, nil
}

// extractPlaylistIDFromURL はURLからプレイリストIDを抽出します
func extractPlaylistIDFromURL(url string) string {
	// "/playlists/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/" の形式からIDを抽出
	re := regexp.MustCompile(`/playlists/([a-f0-9\-]+)/?`)
	matches := re.FindStringSubmatch(url)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// Helper functions for safe type conversion

// getStringFromMap はmap[string]interface{}から文字列値を安全に取得するヘルパー関数
func getStringFromMap(m map[string]interface{}, key string) string {
	if value, ok := m[key].(string); ok {
		return value
	}
	return ""
}

// getStringArrayFromMap はmap[string]interface{}から文字列配列を安全に取得するヘルパー関数
func getStringArrayFromMap(m map[string]interface{}, key string) []string {
	if value, ok := m[key].([]interface{}); ok {
		var result []string
		for _, item := range value {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	}
	return []string{}
}

// convertToTOCItems は任意のinterfaceをTableOfContentsItemの配列に変換します
func convertToTOCItems(data interface{}) []TableOfContentsItem {
	if items, ok := data.([]interface{}); ok {
		var result []TableOfContentsItem
		for _, item := range items {
			if itemMap, ok := item.(map[string]interface{}); ok {
				tocItem := TableOfContentsItem{
					ID:       getStringFromMap(itemMap, "id"),
					Title:    getStringFromMap(itemMap, "title"),
					Href:     getStringFromMap(itemMap, "href"),
					Level:    getIntFromMap(itemMap, "level"),
					Parent:   getStringFromMap(itemMap, "parent"),
					Metadata: itemMap,
				}
				result = append(result, tocItem)
			}
		}
		return result
	}
	return []TableOfContentsItem{}
}

// getIntFromMap はmap[string]interface{}から整数値を安全に取得するヘルパー関数
func getIntFromMap(m map[string]interface{}, key string) int {
	if value, ok := m[key].(float64); ok {
		return int(value)
	}
	if value, ok := m[key].(int); ok {
		return value
	}
	return 0
}

// Placeholder functions for large methods that need to be implemented
// These would need the full implementation from the original file

// GetPlaylistsFromPlaylistsPage はプレイリストページからプレイリスト一覧を取得します
func (bc *BrowserClient) GetPlaylistsFromPlaylistsPage() ([]map[string]interface{}, error) {
	log.Printf("プレイリストページからプレイリスト一覧を取得します")
	
	// TODO: Implement the full playlist extraction logic
	// This is a complex function that requires the full implementation
	return []map[string]interface{}{}, fmt.Errorf("GetPlaylistsFromPlaylistsPage not yet implemented in refactored version")
}

// CreatePlaylist は新しいプレイリストを作成します
func (bc *BrowserClient) CreatePlaylist(name, description string, isPublic bool) (map[string]interface{}, error) {
	log.Printf("新しいプレイリストを作成します: %s", name)
	
	// TODO: Implement the full playlist creation logic
	return map[string]interface{}{}, fmt.Errorf("CreatePlaylist not yet implemented in refactored version")
}

// AddContentToPlaylist はプレイリストにコンテンツを追加します
func (bc *BrowserClient) AddContentToPlaylist(playlistID, contentID string) error {
	log.Printf("プレイリスト %s にコンテンツ %s を追加します", playlistID, contentID)
	
	// TODO: Implement the full content addition logic
	return fmt.Errorf("AddContentToPlaylist not yet implemented in refactored version")
}

// GetPlaylistDetails はプレイリストの詳細情報を取得します
func (bc *BrowserClient) GetPlaylistDetails(playlistID string) (map[string]interface{}, error) {
	log.Printf("プレイリスト %s の詳細情報を取得します", playlistID)
	
	// TODO: Implement the full playlist details extraction logic
	return map[string]interface{}{}, fmt.Errorf("GetPlaylistDetails not yet implemented in refactored version")
}

// ExtractTableOfContents は指定されたURLからテーブルオブコンテンツを抽出します
func (bc *BrowserClient) ExtractTableOfContents(url string) (*TableOfContentsResponse, error) {
	log.Printf("目次を抽出します: %s", url)
	
	// TODO: Implement the full table of contents extraction logic
	return &TableOfContentsResponse{}, fmt.Errorf("ExtractTableOfContents not yet implemented in refactored version")
}

// SearchInBook は本の中で検索を実行します
func (bc *BrowserClient) SearchInBook(bookID, searchTerm string) ([]map[string]interface{}, error) {
	log.Printf("本 %s 内で検索を実行します: %s", bookID, searchTerm)
	
	// TODO: Implement the full in-book search logic
	return []map[string]interface{}{}, fmt.Errorf("SearchInBook not yet implemented in refactored version")
}