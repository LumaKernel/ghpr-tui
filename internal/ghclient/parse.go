package ghclient

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseDiff parses a unified diff string into structured data.
func ParseDiff(raw string) ParsedDiff {
	var result ParsedDiff
	lines := strings.Split(raw, "\n")
	i := 0

	for i < len(lines) {
		// Look for "diff --git" header
		if !strings.HasPrefix(lines[i], "diff --git ") {
			i++
			continue
		}

		file, nextI := parseFileDiff(lines, i)
		result.Files = append(result.Files, file)
		i = nextI
	}

	return result
}

func parseFileDiff(lines []string, start int) (FileDiff, int) {
	var file FileDiff
	i := start

	// Parse "diff --git a/path b/path"
	diffLine := lines[i]
	parts := strings.SplitN(diffLine, " b/", 2)
	if len(parts) == 2 {
		file.NewPath = parts[1]
	}
	aparts := strings.SplitN(diffLine, " a/", 2)
	if len(aparts) == 2 {
		file.OldPath = strings.SplitN(aparts[1], " b/", 2)[0]
	}
	i++

	// Parse metadata lines until we hit @@ or next diff
	for i < len(lines) {
		line := lines[i]
		switch {
		case strings.HasPrefix(line, "new file mode"):
			file.IsNew = true
			i++
		case strings.HasPrefix(line, "deleted file mode"):
			file.IsDelete = true
			i++
		case strings.HasPrefix(line, "rename from"):
			file.IsRename = true
			i++
		case strings.HasPrefix(line, "rename to"):
			i++
		case strings.HasPrefix(line, "similarity index"):
			i++
		case strings.HasPrefix(line, "index "):
			i++
		case strings.HasPrefix(line, "--- "):
			i++
		case strings.HasPrefix(line, "+++ "):
			i++
		case strings.HasPrefix(line, "Binary files"):
			file.IsBinary = true
			i++
		case strings.HasPrefix(line, "old mode"):
			i++
		case strings.HasPrefix(line, "new mode"):
			i++
		case strings.HasPrefix(line, "@@"):
			// Start of hunks
			goto parseHunks
		case strings.HasPrefix(line, "diff --git"):
			// Next file
			return file, i
		default:
			i++
		}
	}

parseHunks:
	for i < len(lines) && !strings.HasPrefix(lines[i], "diff --git") {
		// parseHunk consumes all lines until it hits @@ or diff --git or EOF,
		// so lines[i] here is always @@ (or loop already exited).
		hunk, nextI := parseHunk(lines, i)
		file.Hunks = append(file.Hunks, hunk)
		i = nextI
	}

	return file, i
}

func parseHunk(lines []string, start int) (Hunk, int) {
	hunk := Hunk{Header: lines[start]}
	i := start + 1

	// Parse line numbers from header: @@ -oldStart,oldCount +newStart,newCount @@
	oldNum, newNum := parseHunkHeader(lines[start])

	for i < len(lines) {
		line := lines[i]
		if strings.HasPrefix(line, "diff --git") || strings.HasPrefix(line, "@@") {
			break
		}

		var dl DiffLine
		switch {
		case strings.HasPrefix(line, "+"):
			dl = DiffLine{Type: LineAdded, Content: line[1:], NewNum: newNum}
			newNum++
		case strings.HasPrefix(line, "-"):
			dl = DiffLine{Type: LineRemoved, Content: line[1:], OldNum: oldNum}
			oldNum++
		case strings.HasPrefix(line, " "):
			dl = DiffLine{Type: LineContext, Content: line[1:], OldNum: oldNum, NewNum: newNum}
			oldNum++
			newNum++
		case line == `\ No newline at end of file`:
			i++
			continue
		default:
			// Treat as context (empty lines in diff)
			dl = DiffLine{Type: LineContext, Content: line, OldNum: oldNum, NewNum: newNum}
			oldNum++
			newNum++
		}
		hunk.Lines = append(hunk.Lines, dl)
		i++
	}

	return hunk, i
}

func parseHunkHeader(header string) (oldStart, newStart int) {
	// @@ -oldStart,oldCount +newStart,newCount @@ optional text
	header = strings.TrimPrefix(header, "@@ ")
	parts := strings.SplitN(header, " @@", 2)
	// SplitN always returns at least 1 element
	ranges := strings.Fields(parts[0])
	for _, r := range ranges {
		if strings.HasPrefix(r, "-") {
			nums := strings.SplitN(r[1:], ",", 2)
			n, err := strconv.Atoi(nums[0])
			if err == nil {
				oldStart = n
			}
		} else if strings.HasPrefix(r, "+") {
			nums := strings.SplitN(r[1:], ",", 2)
			n, err := strconv.Atoi(nums[0])
			if err == nil {
				newStart = n
			}
		}
	}
	if oldStart == 0 {
		oldStart = 1
	}
	if newStart == 0 {
		newStart = 1
	}
	return
}

// FilePath returns the display path for a file diff.
func (f FileDiff) FilePath() string {
	if f.IsRename {
		return fmt.Sprintf("%s → %s", f.OldPath, f.NewPath)
	}
	if f.NewPath != "" {
		return f.NewPath
	}
	return f.OldPath
}

// Additions returns the number of added lines.
func (f FileDiff) Additions() int {
	count := 0
	for _, h := range f.Hunks {
		for _, l := range h.Lines {
			if l.Type == LineAdded {
				count++
			}
		}
	}
	return count
}

// Deletions returns the number of deleted lines.
func (f FileDiff) Deletions() int {
	count := 0
	for _, h := range f.Hunks {
		for _, l := range h.Lines {
			if l.Type == LineRemoved {
				count++
			}
		}
	}
	return count
}
