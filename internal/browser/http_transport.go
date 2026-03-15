package browser

import (
	"compress/gzip"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

// GzipTransport is a custom transport that automatically handles gzip decompression
type GzipTransport struct {
	Transport http.RoundTripper
}

// RoundTrip implements the http.RoundTripper interface with automatic gzip decompression
func (g *GzipTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := g.Transport.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	// Check if response is gzip compressed
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return resp, fmt.Errorf("failed to create gzip reader: %w", err)
		}

		// Create a new response with decompressed body
		resp.Body = &gzipReadCloser{
			Reader: gzipReader,
			Closer: resp.Body,
		}

		// Remove Content-Encoding header since we've decompressed
		resp.Header.Del("Content-Encoding")
	}

	return resp, nil
}

// gzipReadCloser wraps a gzip.Reader and ensures both gzip reader and original body are closed
type gzipReadCloser struct {
	io.Reader
	Closer io.Closer
}

func (grc *gzipReadCloser) Close() error {
	// Close the gzip reader first
	if gzipReader, ok := grc.Reader.(*gzip.Reader); ok {
		if err := gzipReader.Close(); err != nil {
			slog.Warn("gzipリーダーのクローズに失敗", "error", err)
		}
	}

	// Then close the original response body
	if grc.Closer != nil {
		return grc.Closer.Close()
	}

	return nil
}
