package main

import (
	"os"
	"testing"
	"time"
)

func TestFindSystemChrome(t *testing.T) {
	// Chrome が見つかった場合はファイルが存在すること、見つからない場合はエラーが返ること
	path, err := findSystemChrome()
	if err != nil {
		// Chrome が見つからない場合はエラーが返ること (これは正常)
		t.Logf("Chrome not found (expected in some environments): %v", err)
		return
	}

	// 見つかった場合はファイルが存在すること
	if _, statErr := os.Stat(path); statErr != nil {
		t.Errorf("findSystemChrome() returned %q but file does not exist: %v", path, statErr)
	}
}

func TestWaitForCDPWithTimeout_Timeout(t *testing.T) {
	// 使用されていないポートに接続を試み、タイムアウトすることを確認
	// ポート 59998 は通常使用されていない
	wsURL, err := waitForCDPWithTimeout("59998", 2*time.Second)
	if err == nil {
		t.Error("waitForCDPWithTimeout() should return error when port is not available")
	}
	if wsURL != "" {
		t.Errorf("waitForCDPWithTimeout() should return empty string on timeout, got %q", wsURL)
	}
}
