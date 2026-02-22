package git

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
func (g *GitDiffProvider) GetDiff(_ string, _ DiffOptions) (*DiffResult, error) {
	// TODO: implement
	return nil, nil
}

// parseDiffStat parses git diff --numstat output.
func parseDiffStat(_ string) map[string]fileStat {
	// TODO: implement
	return nil
}

// parseDiffNameStatus parses git diff --name-status output.
func parseDiffNameStatus(_ string) []nameStatusEntry {
	// TODO: implement
	return nil
}
