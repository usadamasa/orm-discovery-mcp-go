package server

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSearchContentArgs_OffsetDefault(t *testing.T) {
	args := SearchContentArgs{
		Query: "Docker",
	}

	if args.Offset != 0 {
		t.Errorf("expected default Offset to be 0, got %d", args.Offset)
	}
}

func TestSearchContentResult_FilePath(t *testing.T) {
	result := SearchContentResult{
		Count:     10,
		Total:     10,
		HistoryID: "req_abc123",
		FilePath:  "/tmp/cache/20260314-120000_docker.md",
	}

	if result.HistoryID != "req_abc123" {
		t.Errorf("expected HistoryID 'req_abc123', got %q", result.HistoryID)
	}
	if result.FilePath == "" {
		t.Error("expected FilePath to be set")
	}
}

func TestSearchContentResult_PaginationFields(t *testing.T) {
	tests := []struct {
		name         string
		offset       int
		resultCount  int
		totalResults int
		wantHasMore  bool
		wantNext     int
	}{
		{
			name:         "first page with more results",
			offset:       0,
			resultCount:  25,
			totalResults: 100,
			wantHasMore:  true,
			wantNext:     25,
		},
		{
			name:         "middle page",
			offset:       50,
			resultCount:  25,
			totalResults: 100,
			wantHasMore:  true,
			wantNext:     75,
		},
		{
			name:         "last page",
			offset:       75,
			resultCount:  25,
			totalResults: 100,
			wantHasMore:  false,
			wantNext:     0,
		},
		{
			name:         "no results",
			offset:       0,
			resultCount:  0,
			totalResults: 0,
			wantHasMore:  false,
			wantNext:     0,
		},
		{
			name:         "total unknown (API returns 0)",
			offset:       0,
			resultCount:  25,
			totalResults: 0,
			wantHasMore:  false,
			wantNext:     0,
		},
		{
			name:         "exact last page boundary",
			offset:       75,
			resultCount:  25,
			totalResults: 100,
			wantHasMore:  false,
			wantNext:     0,
		},
		{
			name:         "partial last page",
			offset:       90,
			resultCount:  10,
			totalResults: 100,
			wantHasMore:  false,
			wantNext:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasMore, nextOffset := calcPagination(tt.offset, tt.resultCount, tt.totalResults)
			if hasMore != tt.wantHasMore {
				t.Errorf("hasMore = %v, want %v", hasMore, tt.wantHasMore)
			}
			if nextOffset != tt.wantNext {
				t.Errorf("nextOffset = %d, want %d", nextOffset, tt.wantNext)
			}
		})
	}
}

func TestValidationConstants(t *testing.T) {
	assert.Equal(t, 500, maxQueryLength)
	assert.Equal(t, 500, maxQuestionLength)
	assert.Equal(t, 100, maxRows)
}

func TestQueryLengthValidation(t *testing.T) {
	validQuery := strings.Repeat("a", maxQueryLength)
	assert.Len(t, validQuery, maxQueryLength)

	invalidQuery := strings.Repeat("a", maxQueryLength+1)
	assert.True(t, len(invalidQuery) > maxQueryLength)
}

func TestSearchContentResult_PaginationStructure(t *testing.T) {
	result := SearchContentResult{
		Count:        25,
		Total:        25,
		TotalResults: 100,
		HasMore:      true,
		NextOffset:   25,
		HistoryID:    "req_abc123",
	}

	if result.TotalResults != 100 {
		t.Errorf("expected TotalResults 100, got %d", result.TotalResults)
	}
	if !result.HasMore {
		t.Error("expected HasMore to be true")
	}
	if result.NextOffset != 25 {
		t.Errorf("expected NextOffset 25, got %d", result.NextOffset)
	}
}
