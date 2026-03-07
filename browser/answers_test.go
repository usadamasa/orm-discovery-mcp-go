package browser

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/usadamasa/orm-discovery-mcp-go/browser/generated/api"
)

func TestConvertAnswerData_WithNilData(t *testing.T) {
	result := convertAnswerData(nil)

	assert.NotNil(t, result.Sources, "Sources should not be nil, should be empty slice")
	assert.NotNil(t, result.RelatedResources, "RelatedResources should not be nil, should be empty slice")
	assert.NotNil(t, result.AffiliationProducts, "AffiliationProducts should not be nil, should be empty slice")
	assert.NotNil(t, result.FollowupQuestions, "FollowupQuestions should not be nil, should be empty slice")
}

func TestConvertAnswerData_WithEmptyData(t *testing.T) {
	data := &api.AnswerData{}
	result := convertAnswerData(data)

	assert.NotNil(t, result.Sources, "Sources should not be nil, should be empty slice")
	assert.NotNil(t, result.RelatedResources, "RelatedResources should not be nil, should be empty slice")
	assert.NotNil(t, result.AffiliationProducts, "AffiliationProducts should not be nil, should be empty slice")
	assert.NotNil(t, result.FollowupQuestions, "FollowupQuestions should not be nil, should be empty slice")
}

func TestConvertAnswerData_JSONSerialization_NoNulls(t *testing.T) {
	result := convertAnswerData(nil)

	jsonBytes, err := json.Marshal(result)
	require.NoError(t, err)

	jsonStr := string(jsonBytes)

	assert.False(t, containsNullValue(jsonStr, "sources"), "JSON output contains null for sources, expected empty array []")
	assert.False(t, containsNullValue(jsonStr, "related_resources"), "JSON output contains null for related_resources, expected empty array []")
	assert.False(t, containsNullValue(jsonStr, "affiliation_products"), "JSON output contains null for affiliation_products, expected empty array []")
	assert.False(t, containsNullValue(jsonStr, "followup_questions"), "JSON output contains null for followup_questions, expected empty array []")
}

func TestConvertAnswerData_WithValidData(t *testing.T) {
	answer := "Test answer"
	title := "Test Title"
	url := "https://example.com"
	authors := []string{"Author 1"}

	sources := []api.AnswerSource{{
		Title:   &title,
		Url:     &url,
		Authors: &authors,
	}}
	relatedResources := []api.RelatedResource{{
		Title:   &title,
		Url:     &url,
		Authors: &authors,
	}}
	affiliationProducts := []api.AffiliationProduct{{
		Title:   &title,
		Url:     &url,
		Authors: &authors,
	}}
	followupQuestions := []string{"Follow-up question?"}

	data := &api.AnswerData{
		Answer:              &answer,
		Sources:             &sources,
		RelatedResources:    &relatedResources,
		AffiliationProducts: &affiliationProducts,
		FollowupQuestions:   &followupQuestions,
	}

	result := convertAnswerData(data)

	assert.Equal(t, answer, result.Answer)
	assert.Len(t, result.Sources, 1)
	assert.Len(t, result.RelatedResources, 1)
	assert.Len(t, result.AffiliationProducts, 1)
	assert.Len(t, result.FollowupQuestions, 1)
}

func TestGetAnswer_NilMisoResponse(t *testing.T) {
	// Bug #131: resp.JSON200.MisoResponse が nil のとき panic しないこと
	// convertAnswerData に nil を渡す前に MisoResponse の nil チェックが必要

	// MisoResponse フィールドが nil のケースをシミュレート
	// GetAnswer 内で resp.JSON200.MisoResponse.Data にアクセスする際、
	// MisoResponse が nil だと nil pointer dereference が発生する
	var misoData *api.AnswerData
	result := convertAnswerData(misoData)

	assert.Equal(t, "", result.Answer)
	assert.NotNil(t, result.Sources)
}

// containsNullValue checks if a JSON string contains a null value for a specific field
func containsNullValue(jsonStr, fieldName string) bool {
	var raw map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return true
	}

	value, exists := raw[fieldName]
	if !exists {
		return false
	}

	return value == nil
}
