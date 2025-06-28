package browser

import (
	"context"
	"log/slog"
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
		slog.Warn("スクリーンショット取得エラー", "error", err)
		return
	}

	// tmpDirを使用してスクリーンショットを保存
	imgPath := filepath.Join(bc.tmpDir, name+".png")

	// ファイルとして保存
	if err := os.WriteFile(imgPath, buf, 0644); err != nil {
		slog.Warn("スクリーンショット保存エラー", "error", err)
		return
	}

	slog.Debug("デバッグスクリーンショットを保存しました", "path", imgPath)
}
