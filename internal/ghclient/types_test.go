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
