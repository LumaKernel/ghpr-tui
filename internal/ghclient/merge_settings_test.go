package ghclient

import (
	"reflect"
	"testing"
)

func TestNewClient(t *testing.T) {
	c := NewClient("owner/repo")
	if c.repo != "owner/repo" {
		t.Errorf("repo = %q, want %q", c.repo, "owner/repo")
	}
}

func TestNewClient_Empty(t *testing.T) {
	c := NewClient("")
	if c.repo != "" {
		t.Errorf("repo = %q, want empty", c.repo)
	}
}

func TestGhArgs_WithRepo(t *testing.T) {
	c := NewClient("owner/repo")
	got := c.ghArgs("pr", "list")
	want := []string{"pr", "list", "--repo", "owner/repo"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ghArgs = %v, want %v", got, want)
	}
}

func TestGhArgs_WithoutRepo(t *testing.T) {
	c := NewClient("")
	got := c.ghArgs("pr", "list")
	want := []string{"pr", "list"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ghArgs = %v, want %v", got, want)
	}
}

func TestAllowedMethods(t *testing.T) {
	tests := []struct {
		name     string
		settings MergeSettings
		want     []string
	}{
		{
			name:     "all allowed",
			settings: MergeSettings{AllowSquash: true, AllowMerge: true, AllowRebase: true},
			want:     []string{"squash", "merge", "rebase"},
		},
		{
			name:     "squash only",
			settings: MergeSettings{AllowSquash: true},
			want:     []string{"squash"},
		},
		{
			name:     "merge and rebase",
			settings: MergeSettings{AllowMerge: true, AllowRebase: true},
			want:     []string{"merge", "rebase"},
		},
		{
			name:     "none allowed",
			settings: MergeSettings{},
			want:     nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.settings.AllowedMethods()
			if len(got) != len(tt.want) {
				t.Fatalf("AllowedMethods() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("AllowedMethods()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
