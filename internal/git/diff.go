package git

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// DiffOptions holds options for diff retrieval.
type DiffOptions struct {
	BaseBranch string // Target branch for comparison (default: "main")
	Staged     bool   // If true, show only staged changes
	Context    int    // Number of context lines in unified diff
}

// FileDiff represents diff information for a single file.
type FileDiff struct {
	Path      string `json:"path"`
	OldPath   string `json:"old_path,omitempty"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	IsBinary  bool   `json:"is_binary"`
	Patch     string `json:"patch,omitempty"`
}

// DiffResult represents the overall diff result.
type DiffResult struct {
	Files          []FileDiff `json:"files"`
	TotalAdditions int        `json:"total_additions"`
	TotalDeletions int        `json:"total_deletions"`
	BaseBranch     string     `json:"base_branch"`
}

// DiffProvider is an interface for retrieving git diffs.
type DiffProvider interface {
	GetDiff(repoPath string, opts DiffOptions) (*DiffResult, error)
}

// fileStat holds parsed numstat information for a file.
type fileStat struct {
	additions int
	deletions int
	isBinary  bool
}

// nameStatusEntry holds parsed name-status information for a file.
type nameStatusEntry struct {
	status  string
	path    string
	oldPath string
}

// GitDiffProvider implements DiffProvider using git commands.
type GitDiffProvider struct{}

// NewGitDiffProvider creates a new GitDiffProvider.
func NewGitDiffProvider() *GitDiffProvider {
	return &GitDiffProvider{}
}

// GetDiff retrieves diff information for the given repository.
func (g *GitDiffProvider) GetDiff(repoPath string, opts DiffOptions) (*DiffResult, error) {
	baseBranch := opts.BaseBranch
	if baseBranch == "" {
		baseBranch = "main"
	}

	// Build diff arguments based on options
	diffRef := buildDiffRef(baseBranch, opts.Staged)

	// Get numstat output
	numstatArgs := append([]string{"diff", "--numstat"}, diffRef...)
	numstatOut, err := runGitCommand(repoPath, numstatArgs)
	if err != nil {
		return nil, fmt.Errorf("git diff --numstat: %w", err)
	}

	// Get name-status output
	nameStatusArgs := append([]string{"diff", "--name-status"}, diffRef...)
	nameStatusOut, err := runGitCommand(repoPath, nameStatusArgs)
	if err != nil {
		return nil, fmt.Errorf("git diff --name-status: %w", err)
	}

	// Get unified diff (patch text)
	patchArgs := []string{"diff"}
	if opts.Context > 0 {
		patchArgs = append(patchArgs, fmt.Sprintf("-U%d", opts.Context))
	}
	patchArgs = append(patchArgs, diffRef...)
	patchOut, err := runGitCommand(repoPath, patchArgs)
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}

	stats := parseDiffStat(numstatOut)
	entries := parseDiffNameStatus(nameStatusOut)
	patches := splitPatchByFile(patchOut)

	result := &DiffResult{
		BaseBranch: baseBranch,
	}

	for _, entry := range entries {
		fd := FileDiff{
			Path:    entry.path,
			OldPath: entry.oldPath,
			Status:  entry.status,
		}

		// Look up stat by path (try both path and oldPath for renames)
		statKey := entry.path
		if entry.oldPath != "" {
			// For renames, numstat uses "oldpath => newpath" or just newpath
			renameKey := entry.oldPath + " => " + entry.path
			if s, ok := stats[renameKey]; ok {
				fd.Additions = s.additions
				fd.Deletions = s.deletions
				fd.IsBinary = s.isBinary
				statKey = "" // already handled
			}
		}
		if statKey != "" {
			if s, ok := stats[statKey]; ok {
				fd.Additions = s.additions
				fd.Deletions = s.deletions
				fd.IsBinary = s.isBinary
			}
		}

		// Assign patch text for this file
		if p, ok := patches[entry.path]; ok {
			fd.Patch = p
		} else if entry.oldPath != "" {
			if p, ok := patches[entry.oldPath]; ok {
				fd.Patch = p
			}
		}

		result.Files = append(result.Files, fd)
		result.TotalAdditions += fd.Additions
		result.TotalDeletions += fd.Deletions
	}

	return result, nil
}

// buildDiffRef returns the arguments for the diff reference based on options.
func buildDiffRef(baseBranch string, staged bool) []string {
	if staged {
		return []string{"--cached"}
	}
	return []string{baseBranch + "...HEAD"}
}

// runGitCommand executes a git command in the given repository directory.
func runGitCommand(repoPath string, args []string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("running git %v: %w (output: %s)", args, err, string(out))
	}
	return string(out), nil
}

// parseDiffStat parses git diff --numstat output.
// Format: additions\tdeletions\tfilepath
// Binary files show as: -\t-\tfilepath
func parseDiffStat(output string) map[string]fileStat {
	result := make(map[string]fileStat)
	if strings.TrimSpace(output) == "" {
		return result
	}

	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 3 {
			continue
		}

		fs := fileStat{}
		if parts[0] == "-" && parts[1] == "-" {
			fs.isBinary = true
		} else {
			adds, err := strconv.Atoi(parts[0])
			if err == nil {
				fs.additions = adds
			}
			dels, err := strconv.Atoi(parts[1])
			if err == nil {
				fs.deletions = dels
			}
		}
		result[parts[2]] = fs
	}
	return result
}

// parseDiffNameStatus parses git diff --name-status output.
// Format: STATUS\tfilepath (or STATUS\told_path\tnew_path for renames)
// Status codes: A=added, M=modified, D=deleted, R=renamed (Rxx with similarity)
func parseDiffNameStatus(output string) []nameStatusEntry {
	if strings.TrimSpace(output) == "" {
		return nil
	}

	var entries []nameStatusEntry
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}

		entry := nameStatusEntry{}
		statusCode := parts[0]

		switch {
		case statusCode == "A":
			entry.status = "added"
			entry.path = parts[1]
		case statusCode == "M":
			entry.status = "modified"
			entry.path = parts[1]
		case statusCode == "D":
			entry.status = "deleted"
			entry.path = parts[1]
		case strings.HasPrefix(statusCode, "R"):
			entry.status = "renamed"
			if len(parts) >= 3 {
				entry.oldPath = parts[1]
				entry.path = parts[2]
			} else {
				entry.path = parts[1]
			}
		default:
			// Unknown status, include as-is
			entry.status = statusCode
			entry.path = parts[1]
		}

		entries = append(entries, entry)
	}
	return entries
}

// splitPatchByFile splits a unified diff output into per-file patches.
// Each file section starts with "diff --git a/... b/..."
func splitPatchByFile(patchOutput string) map[string]string {
	result := make(map[string]string)
	if strings.TrimSpace(patchOutput) == "" {
		return result
	}

	sections := strings.Split(patchOutput, "diff --git ")
	for _, section := range sections {
		if section == "" {
			continue
		}
		// Re-add the prefix for a complete patch
		fullSection := "diff --git " + section

		// Extract file path from "diff --git a/path b/path"
		firstLine := strings.SplitN(section, "\n", 2)[0]
		parts := strings.SplitN(firstLine, " ", 2)
		if len(parts) < 2 {
			continue
		}

		// Extract b/path part
		bParts := strings.SplitN(parts[1], " ", 2)
		var filePath string
		if len(bParts) >= 1 {
			// Use b/path, stripping "b/" prefix
			path := bParts[0]
			if len(bParts) > 1 {
				path = bParts[1]
			}
			filePath = strings.TrimPrefix(path, "b/")
		}

		if filePath != "" {
			result[filePath] = strings.TrimRight(fullSection, "\n")
		}
	}
	return result
}
