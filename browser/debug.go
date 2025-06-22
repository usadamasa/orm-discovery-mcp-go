package browser

import (
	"context"
	"log"
	"os"

	"github.com/chromedp/chromedp"
)

func (bc *BrowserClient) debugScreenshot(ctx context.Context, name string) {
	var buf []byte
	if err := chromedp.Run(ctx, chromedp.FullScreenshot(&buf, 90)); err != nil {
		log.Fatal(err)
	}

	imgPath := "/Users/usadamasa/src/github.com/usadamasa/orm-discovery-mcp-go/tmp/"

	// ファイルとして保存
	if err := os.WriteFile(imgPath+name+".png", buf, 0644); err != nil {
		log.Fatal(err)
	}
}
