package version

import (
	"runtime/debug"
	"strings"
	"testing"
)

func TestDisplayString(t *testing.T) {
	tests := []struct {
		name string
		info Info
		want string
	}{
		{
			name: "all fields set",
			info: Info{Version: "v1.2.3", Commit: "abc1234", Date: "2024-01-01T00:00:00Z"},
			want: "v1.2.3 (commit: abc1234, built: 2024-01-01T00:00:00Z)",
		},
		{
			name: "default commit and date",
			info: Info{Version: "v0.0.2", Commit: "none", Date: "unknown"},
			want: "v0.0.2",
		},
		{
			name: "only commit available",
			info: Info{Version: "v0.0.2", Commit: "abc1234", Date: "unknown"},
			want: "v0.0.2 (commit: abc1234)",
		},
		{
			name: "only date available",
			info: Info{Version: "v0.0.2", Commit: "none", Date: "2024-01-01T00:00:00Z"},
			want: "v0.0.2 (built: 2024-01-01T00:00:00Z)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.info.DisplayString()
			if got != tt.want {
				t.Errorf("DisplayString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDisplayString_NoParentheses(t *testing.T) {
	info := Info{Version: "v0.0.2", Commit: "none", Date: "unknown"}
	got := info.DisplayString()
	if strings.Contains(got, "(") || strings.Contains(got, ")") {
		t.Errorf("DisplayString() = %q, should not contain parentheses", got)
	}
}

func TestResolve_LdflagsSet(t *testing.T) {
	info := Resolve("v1.2.3", "abc1234", "2024-01-01T00:00:00Z")

	if info.Version != "v1.2.3" {
		t.Errorf("Version = %q, want %q", info.Version, "v1.2.3")
	}
	if info.Commit != "abc1234" {
		t.Errorf("Commit = %q, want %q", info.Commit, "abc1234")
	}
	if info.Date != "2024-01-01T00:00:00Z" {
		t.Errorf("Date = %q, want %q", info.Date, "2024-01-01T00:00:00Z")
	}
}

func TestResolve_DevWithBuildInfo(t *testing.T) {
	original := readBuildInfo
	t.Cleanup(func() { readBuildInfo = original })

	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{
				Version: "v0.3.0",
			},
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "def5678abcdef5678abcdef5678abcdef5678abc"},
				{Key: "vcs.time", Value: "2024-06-15T12:00:00Z"},
			},
		}, true
	}

	info := Resolve("dev", "none", "unknown")

	if info.Version != "v0.3.0" {
		t.Errorf("Version = %q, want %q", info.Version, "v0.3.0")
	}
	if info.Commit != "def5678abcdef5678abcdef5678abcdef5678abc" {
		t.Errorf("Commit = %q, want %q", info.Commit, "def5678abcdef5678abcdef5678abcdef5678abc")
	}
	if info.Date != "2024-06-15T12:00:00Z" {
		t.Errorf("Date = %q, want %q", info.Date, "2024-06-15T12:00:00Z")
	}
}

func TestResolve_DevWithBuildInfoDevel(t *testing.T) {
	original := readBuildInfo
	t.Cleanup(func() { readBuildInfo = original })

	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{
				Version: "(devel)",
			},
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "aaa1111"},
				{Key: "vcs.time", Value: "2024-03-01T09:00:00Z"},
			},
		}, true
	}

	info := Resolve("dev", "none", "unknown")

	if info.Version != "dev" {
		t.Errorf("Version = %q, want %q", info.Version, "dev")
	}
	if info.Commit != "aaa1111" {
		t.Errorf("Commit = %q, want %q", info.Commit, "aaa1111")
	}
	if info.Date != "2024-03-01T09:00:00Z" {
		t.Errorf("Date = %q, want %q", info.Date, "2024-03-01T09:00:00Z")
	}
}

func TestResolve_DevWithoutBuildInfo(t *testing.T) {
	original := readBuildInfo
	t.Cleanup(func() { readBuildInfo = original })

	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return nil, false
	}

	info := Resolve("dev", "none", "unknown")

	if info.Version != "dev" {
		t.Errorf("Version = %q, want %q", info.Version, "dev")
	}
	if info.Commit != "none" {
		t.Errorf("Commit = %q, want %q", info.Commit, "none")
	}
	if info.Date != "unknown" {
		t.Errorf("Date = %q, want %q", info.Date, "unknown")
	}
}
