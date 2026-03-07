package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type testItem struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

func TestAppendAndReadAll(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	items := []testItem{
		{ID: "t-001", Title: "First"},
		{ID: "t-002", Title: "Second"},
	}

	for _, item := range items {
		if err := Append(path, item); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}

	got, err := ReadAll[testItem](path)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got))
	}
	if got[0].ID != "t-001" || got[1].ID != "t-002" {
		t.Errorf("unexpected items: %+v", got)
	}
}

func TestReadAllNonExistent(t *testing.T) {
	got, err := ReadAll[testItem]("/nonexistent/file.jsonl")
	if err != nil {
		t.Fatalf("expected nil error for nonexistent, got: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil, got: %+v", got)
	}
}

func TestRemove(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	for _, item := range []testItem{
		{ID: "t-001", Title: "Keep"},
		{ID: "t-002", Title: "Remove"},
		{ID: "t-003", Title: "Keep"},
	} {
		if err := Append(path, item); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}

	err := Remove(path, func(line json.RawMessage) bool {
		var peek struct {
			ID string `json:"id"`
		}
		json.Unmarshal(line, &peek)
		return peek.ID != "t-002"
	})
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}

	got, err := ReadAll[testItem](path)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got))
	}
	if got[0].ID != "t-001" || got[1].ID != "t-003" {
		t.Errorf("unexpected items: %+v", got)
	}
}

func TestFindByID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	for _, item := range []testItem{
		{ID: "t-001", Title: "First"},
		{ID: "t-002", Title: "Second"},
	} {
		if err := Append(path, item); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}

	raw, err := FindByID(path, "t-002")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if raw == nil {
		t.Fatal("expected to find t-002")
	}

	var item testItem
	json.Unmarshal(raw, &item)
	if item.Title != "Second" {
		t.Errorf("expected Second, got %s", item.Title)
	}

	// Not found
	raw, err = FindByID(path, "t-999")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if raw != nil {
		t.Errorf("expected nil for t-999, got %s", string(raw))
	}
}

func TestMoveToCompleted(t *testing.T) {
	dir := t.TempDir()
	activePath := filepath.Join(dir, "tasks.jsonl")
	donePath := filepath.Join(dir, "tasks.done.jsonl")

	for _, item := range []testItem{
		{ID: "t-001", Title: "First"},
		{ID: "t-002", Title: "Second"},
		{ID: "t-003", Title: "Third"},
	} {
		if err := Append(activePath, item); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}

	err := MoveToCompleted(activePath, donePath, "t-002", func(raw json.RawMessage) (json.RawMessage, error) {
		var item testItem
		if err := json.Unmarshal(raw, &item); err != nil {
			return nil, err
		}
		item.Title = "Second (done)"
		return json.Marshal(item)
	})
	if err != nil {
		t.Fatalf("MoveToCompleted: %v", err)
	}

	// Check active
	active, err := ReadAll[testItem](activePath)
	if err != nil {
		t.Fatalf("ReadAll active: %v", err)
	}
	if len(active) != 2 {
		t.Fatalf("expected 2 active, got %d", len(active))
	}
	if active[0].ID != "t-001" || active[1].ID != "t-003" {
		t.Errorf("unexpected active: %+v", active)
	}

	// Check done
	done, err := ReadAll[testItem](donePath)
	if err != nil {
		t.Fatalf("ReadAll done: %v", err)
	}
	if len(done) != 1 {
		t.Fatalf("expected 1 done, got %d", len(done))
	}
	if done[0].Title != "Second (done)" {
		t.Errorf("expected mutated title, got %s", done[0].Title)
	}
}

func TestMoveToCompletedNotFound(t *testing.T) {
	dir := t.TempDir()
	activePath := filepath.Join(dir, "tasks.jsonl")
	os.WriteFile(activePath, []byte(`{"id":"t-001","title":"x"}`+"\n"), 0644)

	err := MoveToCompleted(activePath, filepath.Join(dir, "done.jsonl"), "t-999", func(raw json.RawMessage) (json.RawMessage, error) {
		return raw, nil
	})
	if err == nil {
		t.Fatal("expected error for not found")
	}
}
