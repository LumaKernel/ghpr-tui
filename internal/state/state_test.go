package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func setupTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	return &Store{
		path: path,
		data: storeData{Repos: make(map[string]*RepoState)},
	}
}

func TestStore_ReadUnread(t *testing.T) {
	s := setupTestStore(t)

	if s.IsRead("owner/repo", 1) {
		t.Error("PR should not be read initially")
	}

	s.MarkRead("owner/repo", 1)
	if !s.IsRead("owner/repo", 1) {
		t.Error("PR should be read after MarkRead")
	}

	s.ToggleRead("owner/repo", 1)
	if s.IsRead("owner/repo", 1) {
		t.Error("PR should be unread after ToggleRead")
	}

	s.ToggleRead("owner/repo", 1)
	if !s.IsRead("owner/repo", 1) {
		t.Error("PR should be read after second ToggleRead")
	}
}

func TestStore_FileReviewed(t *testing.T) {
	s := setupTestStore(t)

	if s.IsFileReviewed("owner/repo", 1, "main.go") {
		t.Error("file should not be reviewed initially")
	}

	s.MarkFileReviewed("owner/repo", 1, "main.go")
	if !s.IsFileReviewed("owner/repo", 1, "main.go") {
		t.Error("file should be reviewed after MarkFileReviewed")
	}

	s.ToggleFileReviewed("owner/repo", 1, "main.go")
	if s.IsFileReviewed("owner/repo", 1, "main.go") {
		t.Error("file should not be reviewed after toggle")
	}
}

func TestStore_MarkAllReviewed(t *testing.T) {
	s := setupTestStore(t)
	paths := []string{"a.go", "b.go", "c.go"}

	s.MarkAllReviewed("owner/repo", 1, paths)
	for _, p := range paths {
		if !s.IsFileReviewed("owner/repo", 1, p) {
			t.Errorf("%s should be reviewed", p)
		}
	}
}

func TestStore_ReviewedFileCount(t *testing.T) {
	s := setupTestStore(t)

	if s.ReviewedFileCount("owner/repo", 1) != 0 {
		t.Error("count should be 0 initially")
	}

	s.MarkFileReviewed("owner/repo", 1, "a.go")
	s.MarkFileReviewed("owner/repo", 1, "b.go")
	if s.ReviewedFileCount("owner/repo", 1) != 2 {
		t.Errorf("count = %d, want 2", s.ReviewedFileCount("owner/repo", 1))
	}
}

func TestStore_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	// Create and populate
	s := &Store{
		path: path,
		data: storeData{Repos: make(map[string]*RepoState)},
	}
	s.MarkRead("owner/repo", 1)
	s.MarkFileReviewed("owner/repo", 1, "main.go")
	if err := s.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("state file not created: %v", err)
	}

	// Load into new store
	s2 := &Store{
		path: path,
		data: storeData{Repos: make(map[string]*RepoState)},
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if err := json.Unmarshal(raw, &s2.data); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if s2.data.Repos == nil {
		s2.data.Repos = make(map[string]*RepoState)
	}

	if !s2.IsRead("owner/repo", 1) {
		t.Error("PR should be read after reload")
	}
	if !s2.IsFileReviewed("owner/repo", 1, "main.go") {
		t.Error("file should be reviewed after reload")
	}
}

func TestStore_MultipleRepos(t *testing.T) {
	s := setupTestStore(t)

	s.MarkRead("repo-a", 1)
	s.MarkRead("repo-b", 2)

	if !s.IsRead("repo-a", 1) {
		t.Error("repo-a PR 1 should be read")
	}
	if s.IsRead("repo-a", 2) {
		t.Error("repo-a PR 2 should not be read")
	}
	if !s.IsRead("repo-b", 2) {
		t.Error("repo-b PR 2 should be read")
	}
}
