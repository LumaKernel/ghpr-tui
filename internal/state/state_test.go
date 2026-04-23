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

func TestNewWithPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	// Create with no existing file
	s, err := NewWithPath(path)
	if err != nil {
		t.Fatalf("NewWithPath failed: %v", err)
	}
	s.MarkRead("repo", 1)
	if err := s.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load existing file
	s2, err := NewWithPath(path)
	if err != nil {
		t.Fatalf("NewWithPath (reload) failed: %v", err)
	}
	if !s2.IsRead("repo", 1) {
		t.Error("should be read after reload")
	}
}

func TestStore_NilMapDeserialization(t *testing.T) {
	// Simulate loading state where PRs map or ReviewedFiles map is nil in JSON
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	// Write JSON with null PRs and null ReviewedFiles
	data := `{"repos":{"repo":{"prs":{"1":{"read":true,"reviewedFiles":null,"lastSeenAt":"2026-01-01T00:00:00Z"}}}}}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := NewWithPath(path)
	if err != nil {
		t.Fatalf("NewWithPath failed: %v", err)
	}

	// Should still work despite nil maps
	if !s.IsRead("repo", 1) {
		t.Error("should be read")
	}
	// Accessing ReviewedFiles should not panic
	if s.IsFileReviewed("repo", 1, "file.go") {
		t.Error("file should not be reviewed")
	}
	// Write new data should work
	s.MarkFileReviewed("repo", 1, "file.go")
	if !s.IsFileReviewed("repo", 1, "file.go") {
		t.Error("file should be reviewed after mark")
	}
}

func TestStore_NilRepoState(t *testing.T) {
	// Simulate repo with null PRs map
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	data := `{"repos":{"repo":{"prs":null}}}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := NewWithPath(path)
	if err != nil {
		t.Fatalf("NewWithPath failed: %v", err)
	}

	// Should handle nil PRs map gracefully
	if s.IsRead("repo", 1) {
		t.Error("should not be read")
	}
	s.MarkRead("repo", 1)
	if !s.IsRead("repo", 1) {
		t.Error("should be read after mark")
	}
}

func TestNewWithPath_NullReposJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	// Write JSON with null repos
	if err := os.WriteFile(path, []byte(`{"repos":null}`), 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := NewWithPath(path)
	if err != nil {
		t.Fatalf("NewWithPath failed: %v", err)
	}
	// Should handle null repos gracefully
	s.MarkRead("repo", 1)
	if !s.IsRead("repo", 1) {
		t.Error("should work after null repos deserialization")
	}
}

func TestNewWithPath_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	// Write invalid JSON — Unmarshal fails, should still work with empty data
	if err := os.WriteFile(path, []byte(`{invalid`), 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := NewWithPath(path)
	if err != nil {
		t.Fatalf("NewWithPath failed: %v", err)
	}
	s.MarkRead("repo", 1)
	if !s.IsRead("repo", 1) {
		t.Error("should work after failed unmarshal")
	}
}

func TestStore_SaveError(t *testing.T) {
	// Save to non-existent directory should fail
	s := &Store{
		path: "/nonexistent/dir/state.json",
		data: storeData{Repos: make(map[string]*RepoState)},
	}
	s.MarkRead("repo", 1)
	if err := s.Save(); err == nil {
		t.Error("Save to non-existent dir should fail")
	}
}

func TestNewStore(t *testing.T) {
	// NewStore uses os.UserConfigDir, which should succeed on CI/local
	s, err := NewStore()
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	// Basic sanity: store should be usable
	s.MarkRead("test-repo", 999)
	if !s.IsRead("test-repo", 999) {
		t.Error("should be read after mark")
	}
}

func TestStore_ReviewedFileCount_WithToggledOff(t *testing.T) {
	s := setupTestStore(t)
	s.MarkFileReviewed("repo", 1, "a.go")
	s.MarkFileReviewed("repo", 1, "b.go")
	s.MarkFileReviewed("repo", 1, "c.go")
	// Toggle off b.go — ReviewedFiles["b.go"] = false
	s.ToggleFileReviewed("repo", 1, "b.go")
	// Count should be 2 (a.go and c.go), not 3
	if got := s.ReviewedFileCount("repo", 1); got != 2 {
		t.Errorf("ReviewedFileCount = %d, want 2", got)
	}
}

func TestNewStore_UserConfigDirError(t *testing.T) {
	// On macOS/Linux, os.UserConfigDir uses HOME. Unsetting it causes an error.
	home := os.Getenv("HOME")
	t.Setenv("HOME", "")
	// Also clear XDG_CONFIG_HOME to ensure UserConfigDir fails
	t.Setenv("XDG_CONFIG_HOME", "")
	_, err := NewStore()
	if err == nil {
		// Some platforms may still succeed; skip if so
		t.Skip("os.UserConfigDir did not fail with empty HOME")
	}
	// Restore HOME for subsequent test cleanup
	_ = home
}

func TestNewStore_MkdirAllError(t *testing.T) {
	// Point HOME to a file (not a directory) so MkdirAll fails
	dir := t.TempDir()
	filePath := filepath.Join(dir, "fakehome")
	if err := os.WriteFile(filePath, []byte("x"), 0o444); err != nil {
		t.Fatal(err)
	}
	// Set HOME to the file path — UserConfigDir will return filePath + "/.config" or similar,
	// and MkdirAll will fail because the parent is a file
	t.Setenv("HOME", filePath)
	t.Setenv("XDG_CONFIG_HOME", "")
	_, err := NewStore()
	if err == nil {
		t.Skip("MkdirAll did not fail as expected")
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
