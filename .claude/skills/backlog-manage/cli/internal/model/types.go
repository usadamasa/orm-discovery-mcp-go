package model

import "encoding/json"

// Task represents a backlog task item.
type Task struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Status      string          `json:"status"`
	Priority    string          `json:"priority"`
	Tags        []string        `json:"tags"`
	Source      string          `json:"source"`
	SourceRef   json.RawMessage `json:"source_ref"`
	GitHubIssue json.RawMessage `json:"github_issue"`
	CreatedAt   string          `json:"created_at"`
	CreatedBy   string          `json:"created_by"`
	UpdatedAt   string          `json:"updated_at"`
	DoneAt      json.RawMessage `json:"done_at"`
	Notes       string          `json:"notes"`
}

// Idea represents a backlog idea item.
type Idea struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Status      string          `json:"status"`
	Tags        []string        `json:"tags"`
	Source      string          `json:"source"`
	SourceRef   json.RawMessage `json:"source_ref"`
	PromotedTo  json.RawMessage `json:"promoted_to"`
	CreatedAt   string          `json:"created_at"`
	CreatedBy   string          `json:"created_by"`
	DoneAt      json.RawMessage `json:"done_at"`
}

// Issue represents a backlog issue item.
type Issue struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Severity    string          `json:"severity"`
	Status      string          `json:"status"`
	Tags        []string        `json:"tags"`
	Source      string          `json:"source"`
	SourceRef   json.RawMessage `json:"source_ref"`
	GitHubIssue json.RawMessage `json:"github_issue"`
	CreatedAt   string          `json:"created_at"`
	CreatedBy   string          `json:"created_by"`
	ResolvedAt  json.RawMessage `json:"resolved_at"`
}

// AuditEntry represents an audit/eval log entry.
type AuditEntry struct {
	ID           string         `json:"id"`
	RunAt        string         `json:"run_at"`
	Score        AuditScore     `json:"score"`
	Findings     []AuditFinding `json:"findings"`
	PatchActions []string       `json:"patch_actions"`
}

// AuditScore holds pass/total counts.
type AuditScore struct {
	Passed int `json:"passed"`
	Total  int `json:"total"`
}

// AuditFinding represents a single check result.
type AuditFinding struct {
	Check   string `json:"check"`
	Status  string `json:"status"`
	Detail  string `json:"detail"`
	Patched bool   `json:"patched"`
}

// NewAuditEntry creates a new AuditEntry with auto-calculated score.
func NewAuditEntry(id string, findings []AuditFinding, patchActions []string) AuditEntry {
	passed := 0
	for _, f := range findings {
		if f.Status == "pass" {
			passed++
		}
	}
	if patchActions == nil {
		patchActions = []string{}
	}
	return AuditEntry{
		ID:           id,
		RunAt:        nowUTC(),
		Score:        AuditScore{Passed: passed, Total: len(findings)},
		Findings:     findings,
		PatchActions: patchActions,
	}
}

// NewTask creates a new Task with defaults.
func NewTask(id, title, description, priority string, tags []string) Task {
	if priority == "" {
		priority = "p2"
	}
	if tags == nil {
		tags = []string{}
	}
	now := nowUTC()
	return Task{
		ID:          id,
		Type:        "task",
		Title:       title,
		Description: description,
		Status:      "active",
		Priority:    priority,
		Tags:        tags,
		Source:      "manual",
		SourceRef:   json.RawMessage("null"),
		GitHubIssue: json.RawMessage("null"),
		CreatedAt:   now,
		CreatedBy:   "manual",
		UpdatedAt:   now,
		DoneAt:      json.RawMessage("null"),
		Notes:       "",
	}
}

// NewIdea creates a new Idea with defaults.
func NewIdea(id, title, description string, tags []string) Idea {
	if tags == nil {
		tags = []string{}
	}
	now := nowUTC()
	return Idea{
		ID:          id,
		Type:        "idea",
		Title:       title,
		Description: description,
		Status:      "active",
		Tags:        tags,
		Source:      "manual",
		SourceRef:   json.RawMessage("null"),
		PromotedTo:  json.RawMessage("null"),
		CreatedAt:   now,
		CreatedBy:   "manual",
		DoneAt:      json.RawMessage("null"),
	}
}

// NewIssue creates a new Issue with defaults.
func NewIssue(id, title, description, severity string, tags []string) Issue {
	if severity == "" {
		severity = "medium"
	}
	if tags == nil {
		tags = []string{}
	}
	now := nowUTC()
	return Issue{
		ID:          id,
		Type:        "issue",
		Title:       title,
		Description: description,
		Severity:    severity,
		Status:      "active",
		Tags:        tags,
		Source:      "manual",
		SourceRef:   json.RawMessage("null"),
		GitHubIssue: json.RawMessage("null"),
		CreatedAt:   now,
		CreatedBy:   "manual",
		ResolvedAt:  json.RawMessage("null"),
	}
}
