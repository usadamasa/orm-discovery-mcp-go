package main

import (
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/usadamasa/orm-discovery-mcp-go/browser"
)

func TestFindSystemChrome(t *testing.T) {
	// Chrome が見つかった場合はファイルが存在すること、見つからない場合はエラーが返ること
	path, err := browser.FindSystemChrome()
	if err != nil {
		// Chrome が見つからない場合はエラーが返ること (これは正常)
		t.Logf("Chrome not found (expected in some environments): %v", err)
		return
	}

	// 見つかった場合はファイルが存在すること
	if _, statErr := os.Stat(path); statErr != nil {
		t.Errorf("browser.FindSystemChrome() returned %q but file does not exist: %v", path, statErr)
	}
}

func TestWaitForCDPWithTimeout_Timeout(t *testing.T) {
	// 動的に未使用ポートを取得してすぐに解放する
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("未使用ポートの取得に失敗: %v", err)
	}
	port := strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
	if err := ln.Close(); err != nil { // ポートを解放してから WaitForCDPWithTimeout に渡す
		t.Fatalf("リスナーのクローズに失敗: %v", err)
	}

	wsURL, err := browser.WaitForCDPWithTimeout(port, 2*time.Second)
	if err == nil {
		t.Error("browser.WaitForCDPWithTimeout() should return error when port is not available")
	}
	if wsURL != "" {
		t.Errorf("browser.WaitForCDPWithTimeout() should return empty string on timeout, got %q", wsURL)
	}
}
