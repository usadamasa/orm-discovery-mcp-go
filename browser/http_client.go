package browser

import (
	"bytes"
	"fmt"
	"io"
	"log"

	"net/http"
)

// makeHTTPRequest performs a generic HTTP request and returns the response body
func (bc *BrowserClient) makeHTTPRequest(method, url string, body []byte) ([]byte, error) {
	log.Printf("Making HTTP %s request to: %s", method, url)

	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("User-Agent", bc.userAgent)

	// Add cookies if available
	for _, cookie := range bc.cookies {
		req.AddCookie(cookie)
	}

	resp, err := bc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return respBody, nil
}
