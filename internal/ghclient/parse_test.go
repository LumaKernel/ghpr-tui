package ghclient

import (
	"testing"
)

func TestParseDiff_SingleFile(t *testing.T) {
	raw := `diff --git a/hello.go b/hello.go
index 1234567..abcdef0 100644
--- a/hello.go
+++ b/hello.go
@@ -1,3 +1,4 @@
 package main

+import "fmt"
 func main() {
`
	diff := ParseDiff(raw)
	if len(diff.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(diff.Files))
	}
	f := diff.Files[0]
	if f.OldPath != "hello.go" {
		t.Errorf("OldPath = %q, want %q", f.OldPath, "hello.go")
	}
	if f.NewPath != "hello.go" {
		t.Errorf("NewPath = %q, want %q", f.NewPath, "hello.go")
	}
	if f.Additions() != 1 {
		t.Errorf("Additions = %d, want 1", f.Additions())
	}
	if f.Deletions() != 0 {
		t.Errorf("Deletions = %d, want 0", f.Deletions())
	}
}

func TestParseDiff_MultipleFiles(t *testing.T) {
	raw := `diff --git a/a.go b/a.go
index 1234567..abcdef0 100644
--- a/a.go
+++ b/a.go
@@ -1,2 +1,3 @@
 package a
+var x = 1

diff --git a/b.go b/b.go
new file mode 100644
index 0000000..1234567
--- /dev/null
+++ b/b.go
@@ -0,0 +1,3 @@
+package b
+
+var y = 2
`
	diff := ParseDiff(raw)
	if len(diff.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(diff.Files))
	}

	a := diff.Files[0]
	if a.NewPath != "a.go" {
		t.Errorf("file[0] NewPath = %q, want %q", a.NewPath, "a.go")
	}
	if a.IsNew {
		t.Error("file[0] should not be IsNew")
	}
	if a.Additions() != 1 {
		t.Errorf("file[0] Additions = %d, want 1", a.Additions())
	}

	b := diff.Files[1]
	if b.NewPath != "b.go" {
		t.Errorf("file[1] NewPath = %q, want %q", b.NewPath, "b.go")
	}
	if !b.IsNew {
		t.Error("file[1] should be IsNew")
	}
	if b.Additions() != 3 {
		t.Errorf("file[1] Additions = %d, want 3", b.Additions())
	}
}

func TestParseDiff_DeletedFile(t *testing.T) {
	raw := `diff --git a/old.go b/old.go
deleted file mode 100644
index 1234567..0000000
--- a/old.go
+++ /dev/null
@@ -1,2 +0,0 @@
-package old
-var z = 3
`
	diff := ParseDiff(raw)
	if len(diff.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(diff.Files))
	}
	f := diff.Files[0]
	if !f.IsDelete {
		t.Error("expected IsDelete")
	}
	if f.Additions() != 0 {
		t.Errorf("Additions = %d, want 0", f.Additions())
	}
	if f.Deletions() != 2 {
		t.Errorf("Deletions = %d, want 2", f.Deletions())
	}
}

func TestParseDiff_RenamedFile(t *testing.T) {
	raw := `diff --git a/old.go b/new.go
similarity index 100%
rename from old.go
rename to new.go
`
	diff := ParseDiff(raw)
	if len(diff.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(diff.Files))
	}
	f := diff.Files[0]
	if !f.IsRename {
		t.Error("expected IsRename")
	}
	if f.OldPath != "old.go" {
		t.Errorf("OldPath = %q, want %q", f.OldPath, "old.go")
	}
	if f.NewPath != "new.go" {
		t.Errorf("NewPath = %q, want %q", f.NewPath, "new.go")
	}
	if f.FilePath() != "old.go → new.go" {
		t.Errorf("FilePath = %q, want %q", f.FilePath(), "old.go → new.go")
	}
}

func TestParseDiff_BinaryFile(t *testing.T) {
	raw := `diff --git a/image.png b/image.png
index 1234567..abcdef0 100644
Binary files a/image.png and b/image.png differ
`
	diff := ParseDiff(raw)
	if len(diff.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(diff.Files))
	}
	if !diff.Files[0].IsBinary {
		t.Error("expected IsBinary")
	}
}

func TestParseDiff_NoNewlineAtEnd(t *testing.T) {
	raw := `diff --git a/f.go b/f.go
index 1234567..abcdef0 100644
--- a/f.go
+++ b/f.go
@@ -1 +1 @@
-old
\ No newline at end of file
+new
\ No newline at end of file
`
	diff := ParseDiff(raw)
	if len(diff.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(diff.Files))
	}
	f := diff.Files[0]
	if f.Additions() != 1 {
		t.Errorf("Additions = %d, want 1", f.Additions())
	}
	if f.Deletions() != 1 {
		t.Errorf("Deletions = %d, want 1", f.Deletions())
	}
}

func TestParseHunkHeader(t *testing.T) {
	tests := []struct {
		header   string
		wantOld  int
		wantNew  int
	}{
		{"@@ -1,3 +1,4 @@", 1, 1},
		{"@@ -10,5 +20,8 @@ func foo()", 10, 20},
		{"@@ -0,0 +1,3 @@", 1, 1}, // 0 falls back to 1
		{"@@ -100 +200 @@", 100, 200},
	}
	for _, tt := range tests {
		old, new := parseHunkHeader(tt.header)
		if old != tt.wantOld || new != tt.wantNew {
			t.Errorf("parseHunkHeader(%q) = (%d, %d), want (%d, %d)",
				tt.header, old, new, tt.wantOld, tt.wantNew)
		}
	}
}

func TestParseDiff_HunkLineNumbers(t *testing.T) {
	raw := `diff --git a/f.go b/f.go
index 1234567..abcdef0 100644
--- a/f.go
+++ b/f.go
@@ -5,3 +5,4 @@
 context line
+added line
 another context`
	diff := ParseDiff(raw)
	f := diff.Files[0]
	if len(f.Hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(f.Hunks))
	}
	lines := f.Hunks[0].Lines
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}

	// First line: context at old=5, new=5
	if lines[0].Type != LineContext || lines[0].OldNum != 5 || lines[0].NewNum != 5 {
		t.Errorf("line[0] = %+v, want context at 5/5", lines[0])
	}
	// Second line: added at new=6
	if lines[1].Type != LineAdded || lines[1].NewNum != 6 {
		t.Errorf("line[1] = %+v, want added at new=6", lines[1])
	}
	// Third line: context at old=6, new=7
	if lines[2].Type != LineContext || lines[2].OldNum != 6 || lines[2].NewNum != 7 {
		t.Errorf("line[2] = %+v, want context at 6/7", lines[2])
	}
}

func TestParseDiff_ModeChange(t *testing.T) {
	raw := `diff --git a/script.sh b/script.sh
old mode 100644
new mode 100755
index 1234567..abcdef0
--- a/script.sh
+++ b/script.sh
@@ -1 +1,2 @@
 #!/bin/bash
+echo hello`
	diff := ParseDiff(raw)
	if len(diff.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(diff.Files))
	}
	f := diff.Files[0]
	if f.OldPath != "script.sh" {
		t.Errorf("OldPath = %q, want %q", f.OldPath, "script.sh")
	}
	if f.Additions() != 1 {
		t.Errorf("Additions = %d, want 1", f.Additions())
	}
}

func TestParseDiff_UnknownMetadata(t *testing.T) {
	// Lines that don't match any known prefix should be skipped via default
	raw := `diff --git a/f.go b/f.go
some unknown metadata line
index 1234567..abcdef0
--- a/f.go
+++ b/f.go
@@ -1 +1,2 @@
 existing
+added`
	diff := ParseDiff(raw)
	if len(diff.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(diff.Files))
	}
	if diff.Files[0].Additions() != 1 {
		t.Errorf("Additions = %d, want 1", diff.Files[0].Additions())
	}
}

func TestParseDiff_NoPathInDiffLine(t *testing.T) {
	// Malformed diff line without a/ b/ prefixes
	raw := `diff --git foo bar
--- foo
+++ bar
@@ -1 +1 @@
-old
+new`
	diff := ParseDiff(raw)
	if len(diff.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(diff.Files))
	}
}

func TestParseHunkHeader_Malformed(t *testing.T) {
	// Empty/malformed header should return defaults (1, 1)
	tests := []struct {
		header  string
		wantOld int
		wantNew int
	}{
		{"", 1, 1},
		{"garbage", 1, 1},
		{"@@ @@", 1, 1},
		{"@@ -abc +def @@", 1, 1}, // non-numeric
	}
	for _, tt := range tests {
		old, new := parseHunkHeader(tt.header)
		if old != tt.wantOld || new != tt.wantNew {
			t.Errorf("parseHunkHeader(%q) = (%d, %d), want (%d, %d)",
				tt.header, old, new, tt.wantOld, tt.wantNew)
		}
	}
}

func TestParseHunkHeader_EmptyParts(t *testing.T) {
	// After stripping "@@ ", splitting on " @@" returns empty for just "@@"
	old, new := parseHunkHeader("@@")
	if old != 1 || new != 1 {
		t.Errorf("parseHunkHeader(%q) = (%d, %d), want (1, 1)", "@@", old, new)
	}
}

func TestParseDiff_FileWithNoHunks(t *testing.T) {
	// File diff that ends without any @@ hunk headers (e.g. binary without explicit Binary line)
	raw := `diff --git a/f.bin b/f.bin
index 1234567..abcdef0 100644
`
	diff := ParseDiff(raw)
	if len(diff.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(diff.Files))
	}
	if len(diff.Files[0].Hunks) != 0 {
		t.Errorf("expected 0 hunks, got %d", len(diff.Files[0].Hunks))
	}
}

func TestParseDiff_EmptyInput(t *testing.T) {
	diff := ParseDiff("")
	if len(diff.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(diff.Files))
	}
}

func TestParseDiff_MultipleHunks(t *testing.T) {
	raw := `diff --git a/f.go b/f.go
index 1234567..abcdef0 100644
--- a/f.go
+++ b/f.go
@@ -1,3 +1,3 @@
 line1
-line2
+line2modified
 line3
@@ -10,3 +10,4 @@
 line10
+inserted
 line11
 line12
`
	diff := ParseDiff(raw)
	f := diff.Files[0]
	if len(f.Hunks) != 2 {
		t.Fatalf("expected 2 hunks, got %d", len(f.Hunks))
	}
	if f.Additions() != 2 {
		t.Errorf("Additions = %d, want 2", f.Additions())
	}
	if f.Deletions() != 1 {
		t.Errorf("Deletions = %d, want 1", f.Deletions())
	}
}

func TestParseDiff_FileWithNoHunksFollowedByAnotherFile(t *testing.T) {
	// Tests the "diff --git" case inside metadata parsing (line 79-81)
	// where a file has no hunks and is immediately followed by another diff
	raw := `diff --git a/f.bin b/f.bin
index 1234567..abcdef0 100644
diff --git a/g.go b/g.go
index 1234567..abcdef0 100644
--- a/g.go
+++ b/g.go
@@ -1,1 +1,2 @@
 line1
+added
`
	diff := ParseDiff(raw)
	if len(diff.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(diff.Files))
	}
	if diff.Files[0].NewPath != "f.bin" {
		t.Errorf("first file = %q, want %q", diff.Files[0].NewPath, "f.bin")
	}
	if len(diff.Files[0].Hunks) != 0 {
		t.Errorf("first file should have 0 hunks, got %d", len(diff.Files[0].Hunks))
	}
	if diff.Files[1].NewPath != "g.go" {
		t.Errorf("second file = %q, want %q", diff.Files[1].NewPath, "g.go")
	}
	if len(diff.Files[1].Hunks) != 1 {
		t.Errorf("second file should have 1 hunk, got %d", len(diff.Files[1].Hunks))
	}
}

func TestFileDiff_FilePath(t *testing.T) {
	tests := []struct {
		name string
		file FileDiff
		want string
	}{
		{
			name: "normal file",
			file: FileDiff{OldPath: "f.go", NewPath: "f.go"},
			want: "f.go",
		},
		{
			name: "new file",
			file: FileDiff{NewPath: "new.go", IsNew: true},
			want: "new.go",
		},
		{
			name: "deleted file, only OldPath",
			file: FileDiff{OldPath: "old.go"},
			want: "old.go",
		},
		{
			name: "renamed file",
			file: FileDiff{OldPath: "a.go", NewPath: "b.go", IsRename: true},
			want: "a.go → b.go",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.file.FilePath()
			if got != tt.want {
				t.Errorf("FilePath() = %q, want %q", got, tt.want)
			}
		})
	}
}
