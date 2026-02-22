package model

import "time"

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
