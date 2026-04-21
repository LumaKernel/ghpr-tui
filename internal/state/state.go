package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Store manages persistent review state.
type Store struct {
	path string
	data storeData
}

type storeData struct {
	Repos map[string]*RepoState `json:"repos"`
}

// RepoState holds review state for a single repo.
type RepoState struct {
	PRs map[int]*PRState `json:"prs"`
}

// PRState holds review state for a single PR.
type PRState struct {
	Read          bool            `json:"read"`
	ReviewedFiles map[string]bool `json:"reviewedFiles"`
	LastSeenAt    time.Time       `json:"lastSeenAt"`
}

// NewStore creates or loads a state store.
func NewStore() (*Store, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("config dir: %w", err)
	}
	dir := filepath.Join(configDir, "ghprq")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating state dir: %w", err)
	}
	path := filepath.Join(dir, "state.json")

	s := &Store{
		path: path,
		data: storeData{Repos: make(map[string]*RepoState)},
	}

	raw, err := os.ReadFile(path)
	if err == nil {
		_ = json.Unmarshal(raw, &s.data)
	}
	if s.data.Repos == nil {
		s.data.Repos = make(map[string]*RepoState)
	}

	return s, nil
}

func (s *Store) repoState(repo string) *RepoState {
	rs, ok := s.data.Repos[repo]
	if !ok {
		rs = &RepoState{PRs: make(map[int]*PRState)}
		s.data.Repos[repo] = rs
	}
	if rs.PRs == nil {
		rs.PRs = make(map[int]*PRState)
	}
	return rs
}

func (s *Store) prState(repo string, number int) *PRState {
	rs := s.repoState(repo)
	ps, ok := rs.PRs[number]
	if !ok {
		ps = &PRState{ReviewedFiles: make(map[string]bool)}
		rs.PRs[number] = ps
	}
	if ps.ReviewedFiles == nil {
		ps.ReviewedFiles = make(map[string]bool)
	}
	return ps
}

// IsRead returns whether a PR has been read.
func (s *Store) IsRead(repo string, number int) bool {
	return s.prState(repo, number).Read
}

// MarkRead marks a PR as read.
func (s *Store) MarkRead(repo string, number int) {
	ps := s.prState(repo, number)
	ps.Read = true
	ps.LastSeenAt = time.Now()
}

// ToggleRead toggles the read state of a PR.
func (s *Store) ToggleRead(repo string, number int) {
	ps := s.prState(repo, number)
	ps.Read = !ps.Read
}

// IsFileReviewed returns whether a file has been reviewed.
func (s *Store) IsFileReviewed(repo string, number int, path string) bool {
	return s.prState(repo, number).ReviewedFiles[path]
}

// ToggleFileReviewed toggles the reviewed state of a file.
func (s *Store) ToggleFileReviewed(repo string, number int, path string) {
	ps := s.prState(repo, number)
	ps.ReviewedFiles[path] = !ps.ReviewedFiles[path]
}

// MarkFileReviewed marks a file as reviewed.
func (s *Store) MarkFileReviewed(repo string, number int, path string) {
	ps := s.prState(repo, number)
	ps.ReviewedFiles[path] = true
}

// MarkAllReviewed marks all given files as reviewed.
func (s *Store) MarkAllReviewed(repo string, number int, paths []string) {
	ps := s.prState(repo, number)
	for _, p := range paths {
		ps.ReviewedFiles[p] = true
	}
}

// ReviewedFileCount returns the number of reviewed files for a PR.
func (s *Store) ReviewedFileCount(repo string, number int) int {
	ps := s.prState(repo, number)
	count := 0
	for _, v := range ps.ReviewedFiles {
		if v {
			count++
		}
	}
	return count
}

// Save persists the state to disk.
func (s *Store) Save() error {
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0o644)
}
