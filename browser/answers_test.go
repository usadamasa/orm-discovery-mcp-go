package browser

import (
	"encoding/json"
	"testing"

	"github.com/usadamasa/orm-discovery-mcp-go/browser/generated/api"
)

func TestConvertAnswerData_WithNilData(t *testing.T) {
	result := convertAnswerData(nil)

	if result.Sources == nil {
		t.Error("Sources should not be nil, should be empty slice")
	}
	if result.RelatedResources == nil {
		t.Error("RelatedResources should not be nil, should be empty slice")
	}
	if result.AffiliationProducts == nil {
		t.Error("AffiliationProducts should not be nil, should be empty slice")
	}
	if result.FollowupQuestions == nil {
		t.Error("FollowupQuestions should not be nil, should be empty slice")
	}
}

func TestConvertAnswerData_WithEmptyData(t *testing.T) {
	data := &api.AnswerData{}
	result := convertAnswerData(data)

	if result.Sources == nil {
		t.Error("Sources should not be nil, should be empty slice")
	}
	if result.RelatedResources == nil {
		t.Error("RelatedResources should not be nil, should be empty slice")
	}
	if result.AffiliationProducts == nil {
		t.Error("AffiliationProducts should not be nil, should be empty slice")
	}
	if result.FollowupQuestions == nil {
		t.Error("FollowupQuestions should not be nil, should be empty slice")
	}
}

func TestConvertAnswerData_JSONSerialization_NoNulls(t *testing.T) {
	// Test with nil input
	result := convertAnswerData(nil)

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal AnswerData: %v", err)
	}

	jsonStr := string(jsonBytes)

	// Verify no null values in JSON output
	if containsNullValue(jsonStr, "sources") {
		t.Error("JSON output contains null for sources, expected empty array []")
	}
	if containsNullValue(jsonStr, "related_resources") {
		t.Error("JSON output contains null for related_resources, expected empty array []")
	}
	if containsNullValue(jsonStr, "affiliation_products") {
		t.Error("JSON output contains null for affiliation_products, expected empty array []")
	}
	if containsNullValue(jsonStr, "followup_questions") {
		t.Error("JSON output contains null for followup_questions, expected empty array []")
	}
}

func TestConvertAnswerData_WithValidData(t *testing.T) {
	// Test with actual data to ensure conversion still works
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

	if result.Answer != answer {
		t.Errorf("Expected answer %q, got %q", answer, result.Answer)
	}
	if len(result.Sources) != 1 {
		t.Errorf("Expected 1 source, got %d", len(result.Sources))
	}
	if len(result.RelatedResources) != 1 {
		t.Errorf("Expected 1 related resource, got %d", len(result.RelatedResources))
	}
	if len(result.AffiliationProducts) != 1 {
		t.Errorf("Expected 1 affiliation product, got %d", len(result.AffiliationProducts))
	}
	if len(result.FollowupQuestions) != 1 {
		t.Errorf("Expected 1 followup question, got %d", len(result.FollowupQuestions))
	}
}

// containsNullValue checks if a JSON string contains a null value for a specific field
func containsNullValue(jsonStr, fieldName string) bool {
	// Unmarshal to check for null values
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return true // Conservative: treat parse errors as potential nulls
	}

	value, exists := raw[fieldName]
	if !exists {
		return false // Field doesn't exist, not null
	}

	return value == nil
}
