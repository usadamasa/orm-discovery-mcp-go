package model

import (
	"time"

	"github.com/google/uuid"
)

// Severity is the severity level of a Finding.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityWarning  Severity = "warning"
	SeverityInfo     Severity = "info"
)

// Category is the classification of a Finding.
type Category string

const (
	CategoryMissingTest           Category = "missing_test"
	CategoryInfraChange           Category = "infra_change"
	CategoryLargeDiff             Category = "large_diff"
	CategoryBuildFailure          Category = "build_failure"
	CategoryLintError             Category = "lint_error"
	CategorySecurityVulnerability Category = "security_vulnerability"
	CategoryBreakingChange        Category = "breaking_change"
	CategoryStyleIssue            Category = "style_issue"
)

// Location represents the code location of a finding.
type Location struct {
	FilePath  string `json:"file_path"`
	StartLine int    `json:"start_line,omitempty"`
	EndLine   int    `json:"end_line,omitempty"`
}

// Finding represents a single review finding.
type Finding struct {
	ID         string         `json:"id"`
	Severity   Severity       `json:"severity"`
	Category   Category       `json:"category"`
	Message    string         `json:"message"`
	Location   Location       `json:"location"`
	Confidence float64        `json:"confidence"`
	CriticName string         `json:"critic_name"`
	Suggestion string         `json:"suggestion,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}

// Valid reports whether s is a recognized Severity value.
func (s Severity) Valid() bool {
	switch s {
	case SeverityCritical, SeverityWarning, SeverityInfo:
		return true
	}
	return false
}

// Valid reports whether c is a recognized Category value.
func (c Category) Valid() bool {
	switch c {
	case CategoryMissingTest, CategoryInfraChange, CategoryLargeDiff,
		CategoryBuildFailure, CategoryLintError, CategorySecurityVulnerability,
		CategoryBreakingChange, CategoryStyleIssue:
		return true
	}
	return false
}

// NewFinding creates a new Finding with auto-generated ID, CreatedAt, and default Confidence.
func NewFinding(severity Severity, category Category, message string, location Location) Finding {
	return Finding{
		ID:         "find_" + uuid.New().String()[:8],
		Severity:   severity,
		Category:   category,
		Message:    message,
		Location:   location,
		Confidence: 0.5,
		CreatedAt:  time.Now(),
	}
}
