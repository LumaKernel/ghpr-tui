package ghclient

import "time"

// PR represents a pull request summary from gh pr list.
type PR struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	Author    string    `json:"author"`
	State     string    `json:"state"`
	IsDraft   bool      `json:"isDraft"`
	Additions int       `json:"additions"`
	Deletions int       `json:"deletions"`
	UpdatedAt time.Time `json:"updatedAt"`
	URL       string    `json:"url"`
	HeadRef   string    `json:"headRefName"`
	BaseRef   string    `json:"baseRefName"`
	Labels    []string  `json:"labels"`
}

// prJSON matches the gh CLI JSON output shape.
type prJSON struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	Author    struct {
		Login string `json:"login"`
	} `json:"author"`
	State       string    `json:"state"`
	IsDraft     bool      `json:"isDraft"`
	Additions   int       `json:"additions"`
	Deletions   int       `json:"deletions"`
	UpdatedAt   time.Time `json:"updatedAt"`
	URL         string    `json:"url"`
	HeadRefName string    `json:"headRefName"`
	BaseRefName string    `json:"baseRefName"`
	Labels      []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

func prFromJSON(p prJSON) PR {
	labels := make([]string, len(p.Labels))
	for i, l := range p.Labels {
		labels[i] = l.Name
	}
	return PR{
		Number:    p.Number,
		Title:     p.Title,
		Author:    p.Author.Login,
		State:     p.State,
		IsDraft:   p.IsDraft,
		Additions: p.Additions,
		Deletions: p.Deletions,
		UpdatedAt: p.UpdatedAt,
		URL:       p.URL,
		HeadRef:   p.HeadRefName,
		BaseRef:   p.BaseRefName,
		Labels:    labels,
	}
}

// LineType represents the type of a diff line.
type LineType int

const (
	LineContext LineType = iota
	LineAdded
	LineRemoved
)

// DiffLine represents a single line in a diff hunk.
type DiffLine struct {
	Type    LineType
	Content string
	OldNum  int
	NewNum  int
}

// Hunk represents a diff hunk.
type Hunk struct {
	Header string
	Lines  []DiffLine
}

// FileDiff represents the diff for a single file.
type FileDiff struct {
	OldPath  string
	NewPath  string
	Hunks    []Hunk
	IsBinary bool
	IsNew    bool
	IsDelete bool
	IsRename bool
}

// ParsedDiff is the complete parsed diff for a PR.
type ParsedDiff struct {
	Files []FileDiff
}
