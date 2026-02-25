package git

// FileStatus represents the type of change for a file.
type FileStatus string

const (
	FileStatusAdded    FileStatus = "added"
	FileStatusModified FileStatus = "modified"
	FileStatusDeleted  FileStatus = "deleted"
	FileStatusRenamed  FileStatus = "renamed"
)

// Valid reports whether s is a recognized FileStatus value.
func (s FileStatus) Valid() bool {
	switch s {
	case FileStatusAdded, FileStatusModified, FileStatusDeleted, FileStatusRenamed:
		return true
	}
	return false
}

// ChangedFile represents a file that was changed in a diff.
// Unlike FileDiff, it omits the Patch field for lightweight file listing.
type ChangedFile struct {
	Path      string     `json:"path"`
	OldPath   string     `json:"old_path,omitempty"`
	Status    FileStatus `json:"status"`
	Additions int        `json:"additions"`
	Deletions int        `json:"deletions"`
	IsBinary  bool       `json:"is_binary"`
}

// ExtractChangedFiles converts a DiffResult into a list of ChangedFile.
func ExtractChangedFiles(result *DiffResult) []ChangedFile {
	if len(result.Files) == 0 {
		return nil
	}

	files := make([]ChangedFile, 0, len(result.Files))
	for _, fd := range result.Files {
		files = append(files, ChangedFile{
			Path:      fd.Path,
			OldPath:   fd.OldPath,
			Status:    FileStatus(fd.Status),
			Additions: fd.Additions,
			Deletions: fd.Deletions,
			IsBinary:  fd.IsBinary,
		})
	}
	return files
}
