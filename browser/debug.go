package browser

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

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

	// stateDirのscreenshotsサブディレクトリを使用
	screenshotDir := filepath.Join(bc.stateDir, "screenshots")
	if err := os.MkdirAll(screenshotDir, 0700); err != nil {
		slog.Warn("スクリーンショットディレクトリ作成エラー", "error", err)
		return
	}

	ts := time.Now()
	timestamp := ts.Format("20060102150405") + fmt.Sprintf("%03d", ts.Nanosecond()/1e6)
	imgPath := filepath.Join(screenshotDir, timestamp+"_"+name+".png")

	// ファイルとして保存
	if err := os.WriteFile(imgPath, buf, 0644); err != nil {
		slog.Warn("スクリーンショット保存エラー", "error", err)
		return
	}

	slog.Debug("デバッグスクリーンショットを保存しました", "path", imgPath)
}
