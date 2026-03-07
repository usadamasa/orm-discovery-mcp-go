package browser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/usadamasa/orm-discovery-mcp-go/browser/generated/api"
)

func TestNormalizeSearchResult_ProductIDKey(t *testing.T) {
	// Bug #130: normalizeSearchResult の戻り値に "product_id" キーが含まれること
	// DFS 消費者が product_id で参照するため、id だけでなく product_id も必要
	productID := "9781492077992"
	title := "Learning Go"
	raw := api.RawSearchResult{
		ProductId: &productID,
		Title:     &title,
	}

	result := normalizeSearchResult(raw, 0)

	// "id" キーには値が設定されているはず
	assert.Equal(t, productID, result["id"])

	// "product_id" キーも設定されていること (Bug #130 修正対象)
	pid, ok := result["product_id"]
	assert.True(t, ok, "product_id key should exist in normalized result")
	assert.Equal(t, productID, pid)
}

func TestNormalizeSearchResult_ProductIDKey_Fallback(t *testing.T) {
	// product_id がなく、isbn で fallback する場合も product_id キーが設定されること
	isbn := "978-1-492-07799-2"
	raw := api.RawSearchResult{
		Isbn: &isbn,
	}

	result := normalizeSearchResult(raw, 0)

	assert.Equal(t, isbn, result["id"])

	pid, ok := result["product_id"]
	assert.True(t, ok, "product_id key should exist even with isbn fallback")
	assert.Equal(t, isbn, pid)
}
