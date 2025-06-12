package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
)

func main() {
	// 設定の読み込み
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("設定の読み込みに失敗しました: %v", err)
	}
	log.Printf("設定を読み込みました")

	// 認証情報の確認
	if cfg.OReillyUserID == "" || cfg.OReillyPassword == "" {
		log.Fatalf("OREILLY_USER_ID と OREILLY_PASSWORD が設定されていません")
	}

	// テストモードの確認
	if len(os.Args) > 1 && os.Args[1] == "test" {
		runSearchTest(cfg)
		return
	}

	// O'Reillyクライアントの初期化（ブラウザクライアントを使用）
	log.Printf("ブラウザクライアントを使用してO'Reillyにログインします...")
	
	oreillyClient, err := NewOreillyClientWithBrowser(cfg.OReillyUserID, cfg.OReillyPassword)
	if err != nil {
		log.Fatalf("ブラウザクライアントの初期化に失敗しました: %v", err)
	}
	defer oreillyClient.Close() // プロセス終了時にブラウザをクリーンアップ

	log.Printf("O'Reillyクライアントの初期化が完了しました")
	s := NewServer(oreillyClient)

	if cfg.Transport == "http" {
		log.Printf("HTTPサーバーを起動します :%s/mcp", cfg.Port)
		if err := s.StartStreamableHTTPServer(fmt.Sprintf(":%s", cfg.Port)); err != nil {
			log.Fatalf("HTTPサーバーの起動に失敗しました: %v", err)
		}
	} else {
		log.Printf("サーバーを起動します")
		if err := s.StartStdioServer(); err != nil {
			fmt.Printf("Server error: %v\n", err)
		}
	}
	log.Println("サーバーが正常にシャットダウンしました")
}

// runSearchTest はSearchContentのテストを実行します
func runSearchTest(cfg *Config) {
	log.Printf("SearchContentのテストを開始します")

	// O'Reillyクライアントの初期化
	oreillyClient, err := NewOreillyClientWithBrowser(cfg.OReillyUserID, cfg.OReillyPassword)
	if err != nil {
		log.Fatalf("ブラウザクライアントの初期化に失敗しました: %v", err)
	}
	defer oreillyClient.Close()

	// テスト用の検索クエリ
	testQueries := []string{
		"Go programming",
		"Docker",
		"Python",
	}

	// コマンドライン引数から検索クエリを取得
	if len(os.Args) > 2 {
		if os.Args[2] == "playlists" {
			// プレイリストテストを実行
			runPlaylistTest(oreillyClient)
			return
		}
		testQueries = []string{os.Args[2]}
	}

	for _, query := range testQueries {
		log.Printf("\n=== 検索テスト: %s ===", query)
		
		// SearchParamsを作成
		searchParams := SearchParams{
			Query:        query,
			Rows:         5, // テスト用に少なめに設定
			Languages:    []string{"en", "ja"},
			TzOffset:     -9,
			AiaOnly:      false,
			FeatureFlags: "improveSearchFilters",
			Report:       true,
			IsTopics:     false,
		}

		// 検索を実行
		ctx := context.Background()
		results, err := oreillyClient.Search(ctx, searchParams)
		if err != nil {
			log.Printf("検索エラー: %v", err)
			continue
		}

		// 結果を表示
		log.Printf("検索結果: %d件", len(results.Results))
		
		if len(results.Results) > 0 {
			// 最初の3件を詳細表示
			for i, result := range results.Results {
				if i >= 3 {
					break
				}
				
				log.Printf("\n--- 結果 %d ---", i+1)
				log.Printf("ID: %s", result.ID)
				log.Printf("タイトル: %s", result.Title)
				log.Printf("著者: %v", result.Authors)
				log.Printf("タイプ: %s", result.Type)
				log.Printf("URL: %s", result.URL)
				if result.Description != "" {
					maxLen := 100
					if len(result.Description) < maxLen {
						maxLen = len(result.Description)
					}
					log.Printf("説明: %s", result.Description[:maxLen])
				}
				
				// メタデータをJSON形式で表示
				if result.Metadata != nil {
					metadataJSON, _ := json.MarshalIndent(result.Metadata, "", "  ")
					log.Printf("メタデータ: %s", string(metadataJSON))
				}
			}
		} else {
			log.Printf("検索結果が見つかりませんでした")
		}
		
		log.Printf("=== 検索テスト完了: %s ===\n", query)
	}

	log.Printf("全ての検索テストが完了しました")
}

// runPlaylistTest はプレイリスト機能のテストを実行します
func runPlaylistTest(oreillyClient *OreillyClient) {
	log.Printf("プレイリスト機能のテストを開始します")

	if oreillyClient.browserClient == nil {
		log.Fatalf("ブラウザクライアントが利用できません")
	}

	// 1. プレイリスト一覧の取得テスト
	log.Printf("\n=== プレイリスト一覧取得テスト ===")
	playlists, err := oreillyClient.browserClient.GetPlaylistsFromPlaylistsPage()
	if err != nil {
		log.Printf("プレイリスト一覧取得エラー: %v", err)
	} else {
		log.Printf("プレイリスト一覧取得成功: %d件", len(playlists))
		
		// 最初の3件を詳細表示
		for i, playlist := range playlists {
			if i >= 3 {
				break
			}
			
			log.Printf("\n--- プレイリスト %d ---", i+1)
			if id, ok := playlist["id"].(string); ok {
				log.Printf("ID: %s", id)
			}
			if title, ok := playlist["title"].(string); ok {
				log.Printf("タイトル: %s", title)
			}
			if description, ok := playlist["description"].(string); ok && description != "" {
				log.Printf("説明: %s", description)
			}
			if creator, ok := playlist["creator"].(string); ok && creator != "" {
				log.Printf("作成者: %s", creator)
			}
			if itemCount, ok := playlist["item_count"].(int); ok {
				log.Printf("アイテム数: %d", itemCount)
			}
			if url, ok := playlist["url"].(string); ok && url != "" {
				log.Printf("URL: %s", url)
			}
			if source, ok := playlist["source"].(string); ok {
				log.Printf("ソース: %s", source)
			}
			
			// プレイリスト全体をJSON形式で表示
			playlistJSON, _ := json.MarshalIndent(playlist, "", "  ")
			log.Printf("詳細データ: %s", string(playlistJSON))
		}
	}

	// 2. プレイリスト作成テスト（オプション）
	log.Printf("\n=== プレイリスト作成テスト（スキップ） ===")
	log.Printf("実際の作成テストは手動で実行してください")
	log.Printf("テストコマンド例:")
	log.Printf("  result, err := oreillyClient.browserClient.CreatePlaylist(\"テストプレイリスト\", \"テスト用のプレイリストです\", false)")

	// 3. プレイリスト詳細取得テスト（プレイリストが存在する場合）
	if len(playlists) > 0 {
		log.Printf("\n=== プレイリスト詳細取得テスト ===")
		firstPlaylist := playlists[0]
		if playlistID, ok := firstPlaylist["id"].(string); ok && playlistID != "" {
			log.Printf("プレイリストID「%s」の詳細を取得します", playlistID)
			
			details, err := oreillyClient.browserClient.GetPlaylistDetails(playlistID)
			if err != nil {
				log.Printf("プレイリスト詳細取得エラー: %v", err)
			} else {
				log.Printf("プレイリスト詳細取得成功")
				detailsJSON, _ := json.MarshalIndent(details, "", "  ")
				log.Printf("詳細データ: %s", string(detailsJSON))
			}
		} else {
			log.Printf("有効なプレイリストIDが見つかりませんでした")
		}
	}

	log.Printf("\n=== プレイリスト機能テスト完了 ===")
}
