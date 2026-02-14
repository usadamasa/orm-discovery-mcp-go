package main

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestLearnTechnologyPromptHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		args           map[string]string
		wantErr        bool
		wantErrMsg     string
		wantDescSubstr string
	}{
		{
			name: "basic technology with default level",
			args: map[string]string{
				"technology": "Docker",
			},
			wantErr:        false,
			wantDescSubstr: "Docker",
		},
		{
			name: "technology with beginner level",
			args: map[string]string{
				"technology":       "Kubernetes",
				"experience_level": "beginner",
			},
			wantErr:        false,
			wantDescSubstr: "Kubernetes",
		},
		{
			name: "technology with advanced level",
			args: map[string]string{
				"technology":       "React",
				"experience_level": "advanced",
			},
			wantErr:        false,
			wantDescSubstr: "React",
		},
		{
			name:       "missing required technology argument",
			args:       map[string]string{},
			wantErr:    true,
			wantErrMsg: "technology argument is required",
		},
		{
			name: "empty technology argument",
			args: map[string]string{
				"technology": "",
			},
			wantErr:    true,
			wantErrMsg: "technology argument is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			srv := &Server{}
			req := &mcp.GetPromptRequest{
				Params: &mcp.GetPromptParams{
					Arguments: tc.args,
				},
			}

			result, err := srv.LearnTechnologyPromptHandler(context.Background(), req)

			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil")
					return
				}
				if tc.wantErrMsg != "" && err.Error() != tc.wantErrMsg {
					t.Errorf("expected error message %q but got %q", tc.wantErrMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("expected result but got nil")
				return
			}

			if tc.wantDescSubstr != "" && result.Description == "" {
				t.Errorf("expected description to contain %q but got empty", tc.wantDescSubstr)
			}

			if len(result.Messages) == 0 {
				t.Errorf("expected messages but got empty")
			}
		})
	}
}

func TestResearchTopicPromptHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		args           map[string]string
		wantErr        bool
		wantErrMsg     string
		wantDescSubstr string
	}{
		{
			name: "basic topic with default depth",
			args: map[string]string{
				"topic": "microservices architecture",
			},
			wantErr:        false,
			wantDescSubstr: "microservices",
		},
		{
			name: "topic with detailed depth",
			args: map[string]string{
				"topic": "GraphQL",
				"depth": "detailed",
			},
			wantErr:        false,
			wantDescSubstr: "GraphQL",
		},
		{
			name: "topic with comprehensive depth",
			args: map[string]string{
				"topic": "machine learning",
				"depth": "comprehensive",
			},
			wantErr:        false,
			wantDescSubstr: "machine learning",
		},
		{
			name:       "missing required topic argument",
			args:       map[string]string{},
			wantErr:    true,
			wantErrMsg: "topic argument is required",
		},
		{
			name: "empty topic argument",
			args: map[string]string{
				"topic": "",
			},
			wantErr:    true,
			wantErrMsg: "topic argument is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			srv := &Server{}
			req := &mcp.GetPromptRequest{
				Params: &mcp.GetPromptParams{
					Arguments: tc.args,
				},
			}

			result, err := srv.ResearchTopicPromptHandler(context.Background(), req)

			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil")
					return
				}
				if tc.wantErrMsg != "" && err.Error() != tc.wantErrMsg {
					t.Errorf("expected error message %q but got %q", tc.wantErrMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("expected result but got nil")
				return
			}

			if tc.wantDescSubstr != "" && result.Description == "" {
				t.Errorf("expected description to contain %q but got empty", tc.wantDescSubstr)
			}

			if len(result.Messages) == 0 {
				t.Errorf("expected messages but got empty")
			}
		})
	}
}

func TestDebugErrorPromptHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		args           map[string]string
		wantErr        bool
		wantErrMsg     string
		wantDescSubstr string
	}{
		{
			name: "basic error with required arguments",
			args: map[string]string{
				"error_message": "NullPointerException",
				"technology":    "Java",
			},
			wantErr:        false,
			wantDescSubstr: "Java",
		},
		{
			name: "error with context",
			args: map[string]string{
				"error_message": "CORS error",
				"technology":    "React",
				"context":       "fetching data from backend API",
			},
			wantErr:        false,
			wantDescSubstr: "React",
		},
		{
			name: "missing error_message",
			args: map[string]string{
				"technology": "Python",
			},
			wantErr:    true,
			wantErrMsg: "error_message argument is required",
		},
		{
			name: "missing technology",
			args: map[string]string{
				"error_message": "ImportError",
			},
			wantErr:    true,
			wantErrMsg: "technology argument is required",
		},
		{
			name:       "missing both required arguments",
			args:       map[string]string{},
			wantErr:    true,
			wantErrMsg: "error_message argument is required",
		},
		{
			name: "empty error_message",
			args: map[string]string{
				"error_message": "",
				"technology":    "Go",
			},
			wantErr:    true,
			wantErrMsg: "error_message argument is required",
		},
		{
			name: "empty technology",
			args: map[string]string{
				"error_message": "panic: runtime error",
				"technology":    "",
			},
			wantErr:    true,
			wantErrMsg: "technology argument is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			srv := &Server{}
			req := &mcp.GetPromptRequest{
				Params: &mcp.GetPromptParams{
					Arguments: tc.args,
				},
			}

			result, err := srv.DebugErrorPromptHandler(context.Background(), req)

			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil")
					return
				}
				if tc.wantErrMsg != "" && err.Error() != tc.wantErrMsg {
					t.Errorf("expected error message %q but got %q", tc.wantErrMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("expected result but got nil")
				return
			}

			if tc.wantDescSubstr != "" && result.Description == "" {
				t.Errorf("expected description to contain %q but got empty", tc.wantDescSubstr)
			}

			if len(result.Messages) == 0 {
				t.Errorf("expected messages but got empty")
			}
		})
	}
}

func TestGetStringArg(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		args         map[string]string
		key          string
		defaultValue string
		want         string
	}{
		{
			name:         "existing key with string value",
			args:         map[string]string{"key1": "value1"},
			key:          "key1",
			defaultValue: "default",
			want:         "value1",
		},
		{
			name:         "missing key returns default",
			args:         map[string]string{"key1": "value1"},
			key:          "key2",
			defaultValue: "default",
			want:         "default",
		},
		{
			name:         "nil map returns default",
			args:         nil,
			key:          "key1",
			defaultValue: "default",
			want:         "default",
		},
		{
			name:         "empty map returns default",
			args:         map[string]string{},
			key:          "key1",
			defaultValue: "default",
			want:         "default",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := getStringArg(tc.args, tc.key, tc.defaultValue)
			if got != tc.want {
				t.Errorf("getStringArg() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBuildLearnTechnologyUserMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		technology      string
		experienceLevel string
		wantContains    []string
	}{
		{
			name:            "Docker beginner",
			technology:      "Docker",
			experienceLevel: "beginner",
			wantContains:    []string{"Docker", "beginner", "oreilly_search_content", "oreilly://"},
		},
		{
			name:            "Kubernetes advanced",
			technology:      "Kubernetes",
			experienceLevel: "advanced",
			wantContains:    []string{"Kubernetes", "advanced", "oreilly_search_content", "oreilly://"},
		},
		{
			name:            "React intermediate",
			technology:      "React",
			experienceLevel: "intermediate",
			wantContains:    []string{"React", "intermediate", "oreilly_ask_question"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := buildLearnTechnologyUserMessage(tc.technology, tc.experienceLevel)

			for _, want := range tc.wantContains {
				if !contains(got, want) {
					t.Errorf("buildLearnTechnologyUserMessage() should contain %q, got: %s", want, got)
				}
			}
		})
	}
}

func TestBuildResearchTopicUserMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		topic        string
		depth        string
		wantContains []string
	}{
		{
			name:         "overview depth",
			topic:        "microservices",
			depth:        "overview",
			wantContains: []string{"microservices", "overview", "high-level overview"},
		},
		{
			name:         "detailed depth",
			topic:        "GraphQL",
			depth:        "detailed",
			wantContains: []string{"GraphQL", "detailed", "best practices"},
		},
		{
			name:         "comprehensive depth",
			topic:        "machine learning",
			depth:        "comprehensive",
			wantContains: []string{"machine learning", "comprehensive", "exhaustive research"},
		},
		{
			name:         "default depth (unknown value)",
			topic:        "serverless",
			depth:        "unknown",
			wantContains: []string{"serverless", "high-level overview"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := buildResearchTopicUserMessage(tc.topic, tc.depth)

			for _, want := range tc.wantContains {
				if !contains(got, want) {
					t.Errorf("buildResearchTopicUserMessage() should contain %q, got: %s", want, got)
				}
			}
		})
	}
}

func TestBuildDebugErrorUserMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		errorMessage string
		technology   string
		errorContext string
		wantContains []string
	}{
		{
			name:         "error without context",
			errorMessage: "NullPointerException",
			technology:   "Java",
			errorContext: "",
			wantContains: []string{"NullPointerException", "Java", "oreilly_ask_question", "oreilly_search_content"},
		},
		{
			name:         "error with context",
			errorMessage: "CORS error",
			technology:   "React",
			errorContext: "fetching data from backend API",
			wantContains: []string{"CORS error", "React", "Context:", "fetching data from backend API"},
		},
		{
			name:         "Go panic error",
			errorMessage: "panic: runtime error",
			technology:   "Go",
			errorContext: "when processing large files",
			wantContains: []string{"panic: runtime error", "Go", "Context:", "large files"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := buildDebugErrorUserMessage(tc.errorMessage, tc.technology, tc.errorContext)

			for _, want := range tc.wantContains {
				if !contains(got, want) {
					t.Errorf("buildDebugErrorUserMessage() should contain %q, got: %s", want, got)
				}
			}
		})
	}
}

// contains checks if s contains substr.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && searchSubstring(s, substr)))
}

// searchSubstring performs substring search.
func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
