package browser

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/chromedp/chromedp"
)

func (bc *BrowserClient) debugScreenshot(ctx context.Context, name string) {
	// ORM_MCP_GO_DEBUG環境変数がtrueの場合のみ実行
	if !bc.debug {
		return
	}

	var buf []byte
	if err := chromedp.Run(ctx, chromedp.FullScreenshot(&buf, 90)); err != nil {
		log.Printf("スクリーンショット取得エラー: %v", err)
		return
	}

	// tmpDirを使用してスクリーンショットを保存
	imgPath := filepath.Join(bc.tmpDir, name+".png")

	// ファイルとして保存
	if err := os.WriteFile(imgPath, buf, 0644); err != nil {
		log.Printf("スクリーンショット保存エラー: %v", err)
		return
	}

	log.Printf("デバッグスクリーンショットを保存しました: %s", imgPath)
}
