package store

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Append writes a single JSON line to the file (append-only).
func Append(path string, v any) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	data = append(data, '\n')
	_, err = f.Write(data)
	return err
}

// ReadAll reads all JSONL lines from the file into a slice of T.
func ReadAll[T any](path string) ([]T, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var result []T
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var item T
		if err := json.Unmarshal(line, &item); err != nil {
			return nil, fmt.Errorf("%s line %d: %w", path, lineNum, err)
		}
		result = append(result, item)
	}
	return result, scanner.Err()
}

// Remove rewrites the file keeping only lines for which keepFn returns true.
// Uses tmpfile + rename pattern (grep -v equivalent).
func Remove(path string, keepFn func(json.RawMessage) bool) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	tmpPath := path + ".tmp"
	tmp, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create tmp: %w", err)
	}
	defer func() {
		tmp.Close()
		os.Remove(tmpPath) // clean up on error
	}()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		if keepFn(json.RawMessage(line)) {
			if _, err := tmp.Write(append(append([]byte(nil), line...), '\n')); err != nil {
				return fmt.Errorf("write tmp: %w", err)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan: %w", err)
	}

	f.Close()
	tmp.Close()
	return os.Rename(tmpPath, path)
}

// FindByID reads a JSONL file and returns the raw JSON line matching the given ID.
func FindByID(path, id string) (json.RawMessage, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var peek struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(line, &peek); err != nil {
			continue
		}
		if peek.ID == id {
			cp := make(json.RawMessage, len(line))
			copy(cp, line)
			return cp, nil
		}
	}
	return nil, scanner.Err()
}

// MoveToCompleted finds an item by ID in activePath, applies mutateFn to it,
// appends the result to donePath, and removes it from activePath.
func MoveToCompleted(activePath, donePath, id string, mutateFn func(json.RawMessage) (json.RawMessage, error)) error {
	raw, err := FindByID(activePath, id)
	if err != nil {
		return err
	}
	if raw == nil {
		return fmt.Errorf("item %s not found in %s", id, filepath.Base(activePath))
	}

	mutated, err := mutateFn(raw)
	if err != nil {
		return fmt.Errorf("mutate: %w", err)
	}

	// Append to done file
	doneF, err := os.OpenFile(donePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open done: %w", err)
	}
	if _, err := doneF.Write(append(mutated, '\n')); err != nil {
		doneF.Close()
		return fmt.Errorf("write done: %w", err)
	}
	doneF.Close()

	// Remove from active file
	return Remove(activePath, func(line json.RawMessage) bool {
		var peek struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(line, &peek); err != nil {
			return true
		}
		return peek.ID != id
	})
}
