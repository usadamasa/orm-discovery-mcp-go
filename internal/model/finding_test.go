package model

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFinding_JSONRoundTrip(t *testing.T) {
	now := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name    string
		finding Finding
	}{
		{
			name: "full finding with all fields",
			finding: Finding{
				ID:         "find_abc12345",
				Severity:   SeverityCritical,
				Category:   CategoryMissingTest,
				Message:    "Function Foo has no tests",
				Location:   Location{FilePath: "pkg/foo.go", StartLine: 10, EndLine: 20},
				Confidence: 0.9,
				CriticName: "test-coverage-critic",
				Suggestion: "Add unit tests for Foo",
				Metadata:   map[string]any{"coverage": 0.0},
				CreatedAt:  now,
			},
		},
		{
			name: "minimal finding without optional fields",
			finding: Finding{
				ID:         "find_def67890",
				Severity:   SeverityInfo,
				Category:   CategoryStyleIssue,
				Message:    "Consider using shorter variable name",
				Location:   Location{FilePath: "main.go"},
				Confidence: 0.5,
				CriticName: "style-critic",
				CreatedAt:  now,
			},
		},
		{
			name: "finding with empty metadata",
			finding: Finding{
				ID:         "find_ghi11111",
				Severity:   SeverityWarning,
				Category:   CategoryLargeDiff,
				Message:    "Large diff detected",
				Location:   Location{FilePath: "server.go", StartLine: 1},
				Confidence: 0.7,
				CriticName: "diff-critic",
				CreatedAt:  now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal
			data, err := json.Marshal(tt.finding)
			require.NoError(t, err)

			// Unmarshal
			var got Finding
			err = json.Unmarshal(data, &got)
			require.NoError(t, err)

			// Compare
			assert.Equal(t, tt.finding.ID, got.ID)
			assert.Equal(t, tt.finding.Severity, got.Severity)
			assert.Equal(t, tt.finding.Category, got.Category)
			assert.Equal(t, tt.finding.Message, got.Message)
			assert.Equal(t, tt.finding.Location, got.Location)
			assert.Equal(t, tt.finding.Confidence, got.Confidence)
			assert.Equal(t, tt.finding.CriticName, got.CriticName)
			assert.Equal(t, tt.finding.Suggestion, got.Suggestion)
			assert.True(t, tt.finding.CreatedAt.Equal(got.CreatedAt))
		})
	}
}

func TestFinding_JSONOmitsEmptyOptionalFields(t *testing.T) {
	now := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	f := Finding{
		ID:         "find_omit0001",
		Severity:   SeverityInfo,
		Category:   CategoryStyleIssue,
		Message:    "test",
		Location:   Location{FilePath: "main.go"},
		Confidence: 0.5,
		CriticName: "critic",
		CreatedAt:  now,
		// Suggestion and Metadata are zero values - should be omitted
	}

	data, err := json.Marshal(f)
	require.NoError(t, err)

	jsonStr := string(data)
	assert.NotContains(t, jsonStr, "suggestion")
	assert.NotContains(t, jsonStr, "metadata")
}

func TestSeverity_StringValues(t *testing.T) {
	tests := []struct {
		severity Severity
		expected string
	}{
		{SeverityCritical, "critical"},
		{SeverityWarning, "warning"},
		{SeverityInfo, "info"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.severity))
		})
	}
}

func TestCategory_StringValues(t *testing.T) {
	tests := []struct {
		category Category
		expected string
	}{
		{CategoryMissingTest, "missing_test"},
		{CategoryInfraChange, "infra_change"},
		{CategoryLargeDiff, "large_diff"},
		{CategoryBuildFailure, "build_failure"},
		{CategoryLintError, "lint_error"},
		{CategorySecurityVulnerability, "security_vulnerability"},
		{CategoryBreakingChange, "breaking_change"},
		{CategoryStyleIssue, "style_issue"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.category))
		})
	}
}

func TestSeverity_Valid(t *testing.T) {
	tests := []struct {
		name     string
		severity Severity
		expected bool
	}{
		{"critical is valid", SeverityCritical, true},
		{"warning is valid", SeverityWarning, true},
		{"info is valid", SeverityInfo, true},
		{"empty is invalid", Severity(""), false},
		{"unknown is invalid", Severity("unknown"), false},
		{"uppercase is invalid", Severity("CRITICAL"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.severity.Valid())
		})
	}
}

func TestCategory_Valid(t *testing.T) {
	tests := []struct {
		name     string
		category Category
		expected bool
	}{
		{"missing_test is valid", CategoryMissingTest, true},
		{"infra_change is valid", CategoryInfraChange, true},
		{"large_diff is valid", CategoryLargeDiff, true},
		{"build_failure is valid", CategoryBuildFailure, true},
		{"lint_error is valid", CategoryLintError, true},
		{"security_vulnerability is valid", CategorySecurityVulnerability, true},
		{"breaking_change is valid", CategoryBreakingChange, true},
		{"style_issue is valid", CategoryStyleIssue, true},
		{"empty is invalid", Category(""), false},
		{"unknown is invalid", Category("unknown"), false},
		{"uppercase is invalid", Category("MISSING_TEST"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.category.Valid())
		})
	}
}

func TestNewFinding(t *testing.T) {
	before := time.Now()
	f := NewFinding(SeverityWarning, CategoryLintError, "unused variable", Location{
		FilePath:  "server.go",
		StartLine: 42,
		EndLine:   42,
	})
	after := time.Now()

	// ID starts with "find_" prefix
	assert.True(t, strings.HasPrefix(f.ID, "find_"), "ID should start with 'find_', got %q", f.ID)

	// ID has correct length: "find_" (5) + uuid[:8] (8) = 13
	assert.Len(t, f.ID, 13, "ID should be 13 characters long")

	// IDs are unique
	f2 := NewFinding(SeverityWarning, CategoryLintError, "another issue", Location{FilePath: "main.go"})
	assert.NotEqual(t, f.ID, f2.ID, "each call should generate a unique ID")

	// Fields are set correctly
	assert.Equal(t, SeverityWarning, f.Severity)
	assert.Equal(t, CategoryLintError, f.Category)
	assert.Equal(t, "unused variable", f.Message)
	assert.Equal(t, "server.go", f.Location.FilePath)
	assert.Equal(t, 42, f.Location.StartLine)
	assert.Equal(t, 42, f.Location.EndLine)

	// Default confidence is 0.5
	assert.Equal(t, 0.5, f.Confidence)

	// CreatedAt is set to approximately now
	assert.False(t, f.CreatedAt.IsZero(), "CreatedAt should not be zero")
	assert.True(t, !f.CreatedAt.Before(before), "CreatedAt should be >= before")
	assert.True(t, !f.CreatedAt.After(after), "CreatedAt should be <= after")

	// Optional fields are zero values
	assert.Empty(t, f.CriticName)
	assert.Empty(t, f.Suggestion)
	assert.Nil(t, f.Metadata)
}

func TestLocation_ZeroValue(t *testing.T) {
	loc := Location{}

	assert.Equal(t, "", loc.FilePath)
	assert.Equal(t, 0, loc.StartLine)
	assert.Equal(t, 0, loc.EndLine)

	// JSON marshal of zero Location should omit line numbers
	data, err := json.Marshal(loc)
	require.NoError(t, err)

	jsonStr := string(data)
	assert.NotContains(t, jsonStr, "start_line")
	assert.NotContains(t, jsonStr, "end_line")
}

func TestLocation_PartialValues(t *testing.T) {
	tests := []struct {
		name     string
		loc      Location
		hasStart bool
		hasEnd   bool
	}{
		{
			name:     "only file path",
			loc:      Location{FilePath: "main.go"},
			hasStart: false,
			hasEnd:   false,
		},
		{
			name:     "file path and start line only",
			loc:      Location{FilePath: "main.go", StartLine: 10},
			hasStart: true,
			hasEnd:   false,
		},
		{
			name:     "all fields",
			loc:      Location{FilePath: "main.go", StartLine: 10, EndLine: 20},
			hasStart: true,
			hasEnd:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.loc)
			require.NoError(t, err)

			jsonStr := string(data)
			if tt.hasStart {
				assert.Contains(t, jsonStr, "start_line")
			} else {
				assert.NotContains(t, jsonStr, "start_line")
			}
			if tt.hasEnd {
				assert.Contains(t, jsonStr, "end_line")
			} else {
				assert.NotContains(t, jsonStr, "end_line")
			}
		})
	}
}
