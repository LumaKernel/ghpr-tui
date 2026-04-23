package ghclient

import (
	"testing"
	"time"
)

func TestCheckSummary(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		cs := CheckSummary{}
		if cs.HasChecks() {
			t.Error("empty should not have checks")
		}
		if cs.AllPass() {
			t.Error("empty should not be AllPass")
		}
		if cs.AnyFail() {
			t.Error("empty should not be AnyFail")
		}
	})

	t.Run("all pass", func(t *testing.T) {
		cs := CheckSummary{Total: 3, Pass: 3}
		if !cs.HasChecks() {
			t.Error("should have checks")
		}
		if !cs.AllPass() {
			t.Error("should be AllPass")
		}
		if cs.AnyFail() {
			t.Error("should not AnyFail")
		}
	})

	t.Run("some fail", func(t *testing.T) {
		cs := CheckSummary{Total: 3, Pass: 1, Fail: 2}
		if !cs.AnyFail() {
			t.Error("should AnyFail")
		}
		if cs.AllPass() {
			t.Error("should not AllPass")
		}
	})

	t.Run("pending", func(t *testing.T) {
		cs := CheckSummary{Total: 2, Pass: 1, Pending: 1}
		if cs.AllPass() {
			t.Error("should not AllPass with pending")
		}
		if cs.AnyFail() {
			t.Error("should not AnyFail")
		}
	})
}

func TestComputeCheckSummary(t *testing.T) {
	checks := []statusCheckJSON{
		{Typename: "CheckRun", Conclusion: "SUCCESS"},
		{Typename: "CheckRun", Conclusion: "FAILURE"},
		{Typename: "CheckRun", Conclusion: "SKIPPED"},
		{Typename: "CheckRun", Conclusion: "CANCELLED"},
		{Typename: "CheckRun", Conclusion: ""},        // pending
		{Typename: "CheckRun", Conclusion: "NEUTRAL"},  // counts as pass
		{Typename: "StatusContext", State: "SUCCESS"},
		{Typename: "StatusContext", State: "FAILURE"},
		{Typename: "StatusContext", State: "PENDING"},
	}
	cs := computeCheckSummary(checks)
	if cs.Total != 9 {
		t.Errorf("Total = %d, want 9", cs.Total)
	}
	if cs.Pass != 3 { // SUCCESS + NEUTRAL + StatusContext SUCCESS
		t.Errorf("Pass = %d, want 3", cs.Pass)
	}
	if cs.Fail != 2 { // FAILURE + StatusContext FAILURE
		t.Errorf("Fail = %d, want 2", cs.Fail)
	}
	if cs.Skip != 1 {
		t.Errorf("Skip = %d, want 1", cs.Skip)
	}
	if cs.Cancel != 1 {
		t.Errorf("Cancel = %d, want 1", cs.Cancel)
	}
	if cs.Pending != 2 { // empty conclusion + StatusContext PENDING
		t.Errorf("Pending = %d, want 2", cs.Pending)
	}
}

func TestCheckDuration(t *testing.T) {
	t.Run("completed", func(t *testing.T) {
		c := Check{
			StartedAt:   time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
			CompletedAt: time.Date(2026, 1, 1, 10, 5, 30, 0, time.UTC),
		}
		d := c.Duration()
		if d != 5*time.Minute+30*time.Second {
			t.Errorf("Duration = %v, want 5m30s", d)
		}
	})

	t.Run("not completed", func(t *testing.T) {
		c := Check{
			StartedAt: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
		}
		if c.Duration() != 0 {
			t.Errorf("Duration should be 0 when not completed")
		}
	})
}

func TestParseCheckBucket(t *testing.T) {
	tests := []struct {
		input string
		want  CheckBucket
	}{
		{"pass", CheckBucketPass},
		{"fail", CheckBucketFail},
		{"pending", CheckBucketPending},
		{"skipping", CheckBucketSkip},
		{"cancel", CheckBucketCancel},
		{"unknown", CheckBucketPending},
	}
	for _, tt := range tests {
		got := ParseCheckBucket(tt.input)
		if got != tt.want {
			t.Errorf("ParseCheckBucket(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestCheckBucketType(t *testing.T) {
	c := Check{Bucket: "fail"}
	if c.BucketType() != CheckBucketFail {
		t.Errorf("BucketType() = %d, want %d", c.BucketType(), CheckBucketFail)
	}
}

func TestComputeCheckSummary_EdgeCases(t *testing.T) {
	checks := []statusCheckJSON{
		{Typename: "CheckRun", Conclusion: "TIMED_OUT"},
		{Typename: "CheckRun", Conclusion: "ACTION_REQUIRED"},
		{Typename: "StatusContext", State: "ERROR"},
		{Typename: "StatusContext", State: "EXPECTED"},
		{Typename: "StatusContext", State: "UNKNOWN_STATE"},
		{Typename: "UnknownType"},
	}
	cs := computeCheckSummary(checks)
	if cs.Total != 6 {
		t.Errorf("Total = %d, want 6", cs.Total)
	}
	if cs.Fail != 3 { // TIMED_OUT + ACTION_REQUIRED + ERROR
		t.Errorf("Fail = %d, want 3", cs.Fail)
	}
	if cs.Pending != 3 { // EXPECTED + UNKNOWN_STATE + UnknownType
		t.Errorf("Pending = %d, want 3", cs.Pending)
	}
}

func TestCheckDuration_NoStart(t *testing.T) {
	c := Check{
		CompletedAt: time.Date(2026, 1, 1, 10, 5, 0, 0, time.UTC),
	}
	if c.Duration() != 0 {
		t.Error("Duration should be 0 when no start time")
	}
}

func TestPrFromJSON_NoLabels(t *testing.T) {
	p := prJSON{Number: 1, Title: "No labels"}
	pr := prFromJSON(p)
	if len(pr.Labels) != 0 {
		t.Errorf("Labels = %v, want empty", pr.Labels)
	}
}

func TestGroupCommentThreads(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		threads := GroupCommentThreads(nil)
		if len(threads) != 0 {
			t.Errorf("expected 0 threads, got %d", len(threads))
		}
	})

	t.Run("single root no replies", func(t *testing.T) {
		comments := []ReviewComment{
			{ID: 1, Body: "root comment", Path: "main.go", Line: 10},
		}
		threads := GroupCommentThreads(comments)
		if len(threads) != 1 {
			t.Fatalf("expected 1 thread, got %d", len(threads))
		}
		if threads[0].Root.ID != 1 {
			t.Errorf("root ID = %d, want 1", threads[0].Root.ID)
		}
		if len(threads[0].Replies) != 0 {
			t.Errorf("expected 0 replies, got %d", len(threads[0].Replies))
		}
	})

	t.Run("root with replies", func(t *testing.T) {
		comments := []ReviewComment{
			{ID: 1, Body: "root", Path: "main.go", Line: 10},
			{ID: 2, Body: "reply 1", InReplyToID: 1},
			{ID: 3, Body: "reply 2", InReplyToID: 1},
		}
		threads := GroupCommentThreads(comments)
		if len(threads) != 1 {
			t.Fatalf("expected 1 thread, got %d", len(threads))
		}
		if len(threads[0].Replies) != 2 {
			t.Errorf("expected 2 replies, got %d", len(threads[0].Replies))
		}
	})

	t.Run("multiple threads", func(t *testing.T) {
		comments := []ReviewComment{
			{ID: 10, Body: "root A", Path: "a.go", Line: 1},
			{ID: 20, Body: "root B", Path: "b.go", Line: 5},
			{ID: 30, Body: "reply to A", InReplyToID: 10},
			{ID: 40, Body: "reply to B", InReplyToID: 20},
		}
		threads := GroupCommentThreads(comments)
		if len(threads) != 2 {
			t.Fatalf("expected 2 threads, got %d", len(threads))
		}
		// Threads should be in order of root IDs
		if threads[0].Root.ID != 10 {
			t.Errorf("first thread root ID = %d, want 10", threads[0].Root.ID)
		}
		if threads[1].Root.ID != 20 {
			t.Errorf("second thread root ID = %d, want 20", threads[1].Root.ID)
		}
	})

	t.Run("orphan reply ignored", func(t *testing.T) {
		comments := []ReviewComment{
			{ID: 1, Body: "root", Path: "main.go", Line: 10},
			{ID: 2, Body: "orphan reply", InReplyToID: 999},
		}
		threads := GroupCommentThreads(comments)
		if len(threads) != 1 {
			t.Fatalf("expected 1 thread, got %d", len(threads))
		}
		if len(threads[0].Replies) != 0 {
			t.Errorf("orphan reply should not be attached, got %d replies", len(threads[0].Replies))
		}
	})
}

func TestCommentsForFile(t *testing.T) {
	threads := []CommentThread{
		{Root: ReviewComment{Path: "main.go", Line: 1}},
		{Root: ReviewComment{Path: "util.go", Line: 5}},
		{Root: ReviewComment{Path: "main.go", Line: 10}},
	}

	t.Run("matching file", func(t *testing.T) {
		result := CommentsForFile(threads, "main.go")
		if len(result) != 2 {
			t.Errorf("expected 2 threads for main.go, got %d", len(result))
		}
	})

	t.Run("no match", func(t *testing.T) {
		result := CommentsForFile(threads, "other.go")
		if len(result) != 0 {
			t.Errorf("expected 0 threads for other.go, got %d", len(result))
		}
	})

	t.Run("empty threads", func(t *testing.T) {
		result := CommentsForFile(nil, "main.go")
		if len(result) != 0 {
			t.Errorf("expected 0 threads for nil input, got %d", len(result))
		}
	})
}

func TestPrFromJSON(t *testing.T) {
	p := prJSON{
		Number:      42,
		Title:       "Test PR",
		State:       "OPEN",
		IsDraft:     true,
		Additions:   10,
		Deletions:   5,
		URL:         "https://github.com/test/repo/pull/42",
		HeadRefName: "feature",
		BaseRefName: "main",
	}
	p.Author.Login = "testuser"
	p.Labels = []struct {
		Name string `json:"name"`
	}{{Name: "bug"}, {Name: "urgent"}}
	p.StatusCheckRollup = []statusCheckJSON{
		{Typename: "CheckRun", Conclusion: "SUCCESS"},
	}

	pr := prFromJSON(p)
	if pr.Number != 42 {
		t.Errorf("Number = %d, want 42", pr.Number)
	}
	if pr.Author != "testuser" {
		t.Errorf("Author = %q, want %q", pr.Author, "testuser")
	}
	if !pr.IsDraft {
		t.Error("expected IsDraft")
	}
	if len(pr.Labels) != 2 || pr.Labels[0] != "bug" {
		t.Errorf("Labels = %v, want [bug urgent]", pr.Labels)
	}
	if pr.CheckSummary.Total != 1 || pr.CheckSummary.Pass != 1 {
		t.Errorf("CheckSummary = %+v, want Total=1 Pass=1", pr.CheckSummary)
	}
}
