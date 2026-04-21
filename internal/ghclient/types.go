package ghclient

import "time"

// CheckBucket represents the bucket/status category of a CI check.
type CheckBucket int

const (
	CheckBucketPass CheckBucket = iota
	CheckBucketFail
	CheckBucketPending
	CheckBucketSkip
	CheckBucketCancel
	checkBucketSentinel // exhaustive guard
)

// ParseCheckBucket converts a string bucket to CheckBucket.
func ParseCheckBucket(s string) CheckBucket {
	switch s {
	case "pass":
		return CheckBucketPass
	case "fail":
		return CheckBucketFail
	case "pending":
		return CheckBucketPending
	case "skipping":
		return CheckBucketSkip
	case "cancel":
		return CheckBucketCancel
	default:
		return CheckBucketPending
	}
}

// Check represents a single CI check.
type Check struct {
	Name        string    `json:"name"`
	State       string    `json:"state"`
	Bucket      string    `json:"bucket"`
	Description string    `json:"description"`
	Link        string    `json:"link"`
	Workflow    string    `json:"workflow"`
	Event       string    `json:"event"`
	StartedAt   time.Time `json:"startedAt"`
	CompletedAt time.Time `json:"completedAt"`
}

// BucketType returns the parsed bucket.
func (c Check) BucketType() CheckBucket {
	return ParseCheckBucket(c.Bucket)
}

// Duration returns the check duration, or 0 if not completed.
func (c Check) Duration() time.Duration {
	if c.CompletedAt.IsZero() || c.StartedAt.IsZero() {
		return 0
	}
	return c.CompletedAt.Sub(c.StartedAt)
}

// CheckSummary is a rollup of check statuses.
type CheckSummary struct {
	Total   int
	Pass    int
	Fail    int
	Pending int
	Skip    int
	Cancel  int
}

// HasChecks returns whether there are any checks.
func (cs CheckSummary) HasChecks() bool {
	return cs.Total > 0
}

// AllPass returns whether all checks passed.
func (cs CheckSummary) AllPass() bool {
	return cs.Total > 0 && cs.Pass == cs.Total
}

// AnyFail returns whether any check failed.
func (cs CheckSummary) AnyFail() bool {
	return cs.Fail > 0
}

// PR represents a pull request summary from gh pr list.
type PR struct {
	Number       int          `json:"number"`
	Title        string       `json:"title"`
	Author       string       `json:"author"`
	State        string       `json:"state"`
	IsDraft      bool         `json:"isDraft"`
	Additions    int          `json:"additions"`
	Deletions    int          `json:"deletions"`
	UpdatedAt    time.Time    `json:"updatedAt"`
	URL          string       `json:"url"`
	HeadRef      string       `json:"headRefName"`
	BaseRef      string       `json:"baseRefName"`
	Labels       []string     `json:"labels"`
	CheckSummary CheckSummary
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
	StatusCheckRollup []statusCheckJSON `json:"statusCheckRollup"`
}

type statusCheckJSON struct {
	Typename   string `json:"__typename"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	State      string `json:"state"` // for StatusContext typename
}

func computeCheckSummary(checks []statusCheckJSON) CheckSummary {
	var cs CheckSummary
	cs.Total = len(checks)
	for _, c := range checks {
		switch c.Typename {
		case "CheckRun":
			switch c.Conclusion {
			case "SUCCESS":
				cs.Pass++
			case "FAILURE", "TIMED_OUT", "ACTION_REQUIRED":
				cs.Fail++
			case "CANCELLED":
				cs.Cancel++
			case "SKIPPED":
				cs.Skip++
			case "NEUTRAL":
				cs.Pass++
			default:
				// No conclusion yet = pending/in progress
				cs.Pending++
			}
		case "StatusContext":
			switch c.State {
			case "SUCCESS":
				cs.Pass++
			case "FAILURE", "ERROR":
				cs.Fail++
			case "PENDING", "EXPECTED":
				cs.Pending++
			default:
				cs.Pending++
			}
		default:
			cs.Pending++
		}
	}
	return cs
}

func prFromJSON(p prJSON) PR {
	labels := make([]string, len(p.Labels))
	for i, l := range p.Labels {
		labels[i] = l.Name
	}
	return PR{
		Number:       p.Number,
		Title:        p.Title,
		Author:       p.Author.Login,
		State:        p.State,
		IsDraft:      p.IsDraft,
		Additions:    p.Additions,
		Deletions:    p.Deletions,
		UpdatedAt:    p.UpdatedAt,
		URL:          p.URL,
		HeadRef:      p.HeadRefName,
		BaseRef:      p.BaseRefName,
		Labels:       labels,
		CheckSummary: computeCheckSummary(p.StatusCheckRollup),
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
