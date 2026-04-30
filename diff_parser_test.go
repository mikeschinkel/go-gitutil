package gitutils

import (
	"testing"
)

func TestParseHunkHeader(t *testing.T) {
	tests := []struct {
		name      string
		header    string
		wantOld   int
		wantOldCt int
		wantNew   int
		wantNewCt int
		wantErr   bool
	}{
		{
			name:      "BasicHeader",
			header:    "@@ -10,5 +10,7 @@",
			wantOld:   10,
			wantOldCt: 5,
			wantNew:   10,
			wantNewCt: 7,
			wantErr:   false,
		},
		{
			name:      "HeaderWithFunctionName",
			header:    "@@ -10,5 +10,7 @@ func foo()",
			wantOld:   10,
			wantOldCt: 5,
			wantNew:   10,
			wantNewCt: 7,
			wantErr:   false,
		},
		{
			name:      "SingleLineOld",
			header:    "@@ -10 +10,7 @@",
			wantOld:   10,
			wantOldCt: 1,
			wantNew:   10,
			wantNewCt: 7,
			wantErr:   false,
		},
		{
			name:      "SingleLineNew",
			header:    "@@ -10,5 +10 @@",
			wantOld:   10,
			wantOldCt: 5,
			wantNew:   10,
			wantNewCt: 1,
			wantErr:   false,
		},
		{
			name:      "BothSingleLine",
			header:    "@@ -10 +10 @@",
			wantOld:   10,
			wantOldCt: 1,
			wantNew:   10,
			wantNewCt: 1,
			wantErr:   false,
		},
		{
			name:    "InvalidHeader",
			header:  "not a hunk header",
			wantErr: true,
		},
		{
			name:    "MissingAtSigns",
			header:  "-10,5 +10,7",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hunkHeader, err := ParseHunkHeader(tt.header)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseHunkHeader() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseHunkHeader() unexpected error: %v", err)
				return
			}

			if hunkHeader.OldRange.Start != tt.wantOld {
				t.Errorf("oldStart = %d, want %d", hunkHeader.OldRange.Start, tt.wantOld)
			}
			if hunkHeader.OldRange.Count != tt.wantOldCt {
				t.Errorf("oldCount = %d, want %d", hunkHeader.OldRange.Count, tt.wantOldCt)
			}
			if hunkHeader.NewRange.Start != tt.wantNew {
				t.Errorf("newStart = %d, want %d", hunkHeader.NewRange.Start, tt.wantNew)
			}
			if hunkHeader.NewRange.Count != tt.wantNewCt {
				t.Errorf("newCount = %d, want %d", hunkHeader.NewRange.Count, tt.wantNewCt)
			}
		})
	}
}

func TestParseUnifiedDiff_SingleFile(t *testing.T) {
	diff := `diff --git a/foo.go b/foo.go
index abc123..def456 100644
--- a/foo.go
+++ b/foo.go
@@ -10,4 +10,5 @@ package main
 func foo() {
-    old line
+    new line 1
+    new line 2
     unchanged
 }
`

	fileDiffs, err := ParseUnifiedDiff(diff)
	if err != nil {
		t.Fatalf("ParseUnifiedDiff() error: %v", err)
	}

	if len(fileDiffs) != 1 {
		t.Fatalf("expected 1 file, got %d", len(fileDiffs))
	}

	file := fileDiffs[0]
	if file.OldPath != "foo.go" {
		t.Errorf("OldPath = %q, want %q", file.OldPath, "foo.go")
	}
	if file.NewPath != "foo.go" {
		t.Errorf("NewPath = %q, want %q", file.NewPath, "foo.go")
	}

	if len(file.Hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(file.Hunks))
	}

	hunk := file.Hunks[0]
	if hunk.Header.OldRange.Start != 10 {
		t.Errorf("OldStart = %d, want 10", hunk.Header.OldRange.Start)
	}
	if hunk.Header.OldRange.Count != 4 {
		t.Errorf("OldCount = %d, want 4", hunk.Header.OldRange.Count)
	}
	if hunk.Header.NewRange.Start != 10 {
		t.Errorf("NewStart = %d, want 10", hunk.Header.NewRange.Start)
	}
	if hunk.Header.NewRange.Count != 5 {
		t.Errorf("NewCount = %d, want 5", hunk.Header.NewRange.Count)
	}

	expectedLines := []struct {
		typ     DiffLineType
		content string
	}{
		{DiffContext, "func foo() {"},
		{DiffRemoved, "    old line"},
		{DiffAdded, "    new line 1"},
		{DiffAdded, "    new line 2"},
		{DiffContext, "    unchanged"},
		{DiffContext, "}"},
	}

	if len(hunk.Lines) != len(expectedLines) {
		t.Fatalf("expected %d lines, got %d", len(expectedLines), len(hunk.Lines))
	}

	for i, expected := range expectedLines {
		if hunk.Lines[i].Type != expected.typ {
			t.Errorf("line[%d].Type = %v, want %v", i, hunk.Lines[i].Type, expected.typ)
		}
		if hunk.Lines[i].Content != expected.content {
			t.Errorf("line[%d].Content = %q, want %q", i, hunk.Lines[i].Content, expected.content)
		}
	}
}

func TestParseUnifiedDiff_MultipleFiles(t *testing.T) {
	diff := `diff --git a/file1.go b/file1.go
index abc123..def456 100644
--- a/file1.go
+++ b/file1.go
@@ -1,3 +1,3 @@
 package main
-old
+new
diff --git a/file2.go b/file2.go
index 111222..333444 100644
--- a/file2.go
+++ b/file2.go
@@ -5,2 +5,3 @@
 func bar() {
+    added line
 }
`

	fileDiffs, err := ParseUnifiedDiff(diff)
	if err != nil {
		t.Fatalf("ParseUnifiedDiff() error: %v", err)
	}

	if len(fileDiffs) != 2 {
		t.Fatalf("expected 2 files, got %d", len(fileDiffs))
	}

	if fileDiffs[0].NewPath != "file1.go" {
		t.Errorf("file[0].NewPath = %q, want %q", fileDiffs[0].NewPath, "file1.go")
	}
	if fileDiffs[1].NewPath != "file2.go" {
		t.Errorf("file[1].NewPath = %q, want %q", fileDiffs[1].NewPath, "file2.go")
	}

	if len(fileDiffs[0].Hunks) != 1 {
		t.Errorf("file[0] expected 1 hunk, got %d", len(fileDiffs[0].Hunks))
	}
	if len(fileDiffs[1].Hunks) != 1 {
		t.Errorf("file[1] expected 1 hunk, got %d", len(fileDiffs[1].Hunks))
	}
}

func TestParseUnifiedDiff_NewFile(t *testing.T) {
	diff := `diff --git a/newfile.go b/newfile.go
new file mode 100644
index 0000000..abc123
--- /dev/null
+++ b/newfile.go
@@ -0,0 +1,3 @@
+package main
+
+func new() {}
`

	fileDiffs, err := ParseUnifiedDiff(diff)
	if err != nil {
		t.Fatalf("ParseUnifiedDiff() error: %v", err)
	}

	if len(fileDiffs) != 1 {
		t.Fatalf("expected 1 file, got %d", len(fileDiffs))
	}

	file := fileDiffs[0]
	if file.OldPath != "" {
		t.Errorf("OldPath = %q, want empty (new file)", file.OldPath)
	}
	if file.NewPath != "newfile.go" {
		t.Errorf("NewPath = %q, want %q", file.NewPath, "newfile.go")
	}

	if len(file.Hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(file.Hunks))
	}

	hunk := file.Hunks[0]
	if hunk.Header.OldRange.Start != 0 {
		t.Errorf("OldStart = %d, want 0 (new file)", hunk.Header.OldRange.Start)
	}
	if hunk.Header.NewRange.Start != 1 {
		t.Errorf("NewStart = %d, want 1", hunk.Header.NewRange.Start)
	}
}

func TestParseUnifiedDiff_DeletedFile(t *testing.T) {
	diff := `diff --git a/deleted.go b/deleted.go
deleted file mode 100644
index abc123..0000000
--- a/deleted.go
+++ /dev/null
@@ -1,3 +0,0 @@
-package main
-
-func deleted() {}
`

	fileDiffs, err := ParseUnifiedDiff(diff)
	if err != nil {
		t.Fatalf("ParseUnifiedDiff() error: %v", err)
	}

	if len(fileDiffs) != 1 {
		t.Fatalf("expected 1 file, got %d", len(fileDiffs))
	}

	file := fileDiffs[0]
	if file.OldPath != "deleted.go" {
		t.Errorf("OldPath = %q, want %q", file.OldPath, "deleted.go")
	}
	if file.NewPath != "" {
		t.Errorf("NewPath = %q, want empty (deleted file)", file.NewPath)
	}
}

func TestParseUnifiedDiff_EmptyDiff(t *testing.T) {
	_, err := ParseUnifiedDiff("")
	if err == nil {
		t.Error("ParseUnifiedDiff(\"\") expected error, got nil")
	}
}

func TestComputeHunkChecksum(t *testing.T) {
	hunk := &DiffHunk{
		Lines: []DiffLine{
			{Type: DiffContext, Content: "unchanged"},
			{Type: DiffRemoved, Content: "old line"},
			{Type: DiffAdded, Content: "new line"},
			{Type: DiffContext, Content: "unchanged again"},
		},
	}

	checksum1 := ComputeHunkChecksum(hunk)
	if checksum1 == "" {
		t.Error("ComputeHunkChecksum() returned empty string")
	}

	checksum2 := ComputeHunkChecksum(hunk)
	if checksum1 != checksum2 {
		t.Error("ComputeHunkChecksum() not consistent across calls")
	}

	hunk2 := &DiffHunk{
		Lines: []DiffLine{
			{Type: DiffContext, Content: "unchanged"},
			{Type: DiffRemoved, Content: "old line"},
			{Type: DiffAdded, Content: "new line"},
			{Type: DiffContext, Content: "unchanged again"},
		},
	}

	checksum3 := ComputeHunkChecksum(hunk2)
	if checksum1 != checksum3 {
		t.Error("ComputeHunkChecksum() should be same for identical hunks")
	}

	hunk3 := &DiffHunk{
		Lines: []DiffLine{
			{Type: DiffContext, Content: "unchanged"},
			{Type: DiffRemoved, Content: "different line"},
			{Type: DiffAdded, Content: "new line"},
			{Type: DiffContext, Content: "unchanged again"},
		},
	}

	checksum4 := ComputeHunkChecksum(hunk3)
	if checksum1 == checksum4 {
		t.Error("ComputeHunkChecksum() should differ for different hunks")
	}
}

func TestComputeFileChecksum(t *testing.T) {
	content1 := []byte("package main\n\nfunc main() {}\n")
	checksum1 := ComputeFileChecksum(content1)

	if checksum1 == "" {
		t.Error("ComputeFileChecksum() returned empty string")
	}

	checksum2 := ComputeFileChecksum(content1)
	if checksum1 != checksum2 {
		t.Error("ComputeFileChecksum() not consistent across calls")
	}

	content2 := []byte("package main\n\nfunc main() {}\n")
	checksum3 := ComputeFileChecksum(content2)
	if checksum1 != checksum3 {
		t.Error("ComputeFileChecksum() should be same for identical content")
	}

	content3 := []byte("package main\n\nfunc other() {}\n")
	checksum4 := ComputeFileChecksum(content3)
	if checksum1 == checksum4 {
		t.Error("ComputeFileChecksum() should differ for different content")
	}
}

func TestDiffLineType_String(t *testing.T) {
	tests := []struct {
		typ  DiffLineType
		want string
	}{
		{DiffContext, "context"},
		{DiffAdded, "added"},
		{DiffRemoved, "removed"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.typ.String()
			if got != tt.want {
				t.Errorf("DiffLineType.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractChangeBlocks_AddedLines(t *testing.T) {
	// Test case: 3 lines added at line 11
	fileDiff := &FileDiff{
		NewPath:  "test.go",
		OldLines: []string{"line1", "line2", "line3", "line4", "line5"},
		NewLines: []string{"line1", "line2", "line3", "added1", "added2", "added3", "line4", "line5"},
		Hunks: []*DiffHunk{
			{
				Header: HunkHeader{
					OldRange: HeaderRange{Start: 3, Count: 3},
					NewRange: HeaderRange{Start: 3, Count: 6},
				},
				Lines: []DiffLine{
					{Type: DiffContext, Content: "line3"},
					{Type: DiffAdded, Content: "added1"},
					{Type: DiffAdded, Content: "added2"},
					{Type: DiffAdded, Content: "added3"},
					{Type: DiffContext, Content: "line4"},
				},
			},
		},
	}

	fc := extractChangeBlocks(fileDiff)

	if len(fc.ChangeBlocks) != 1 {
		t.Fatalf("expected 1 change block, got %d", len(fc.ChangeBlocks))
	}

	block := fc.ChangeBlocks[0]
	if block.Type != LinesAdded {
		t.Errorf("Type = %v, want LinesAdded", block.Type)
	}
	if block.CommitRange.Start != 4 || block.CommitRange.Count != 0 {
		t.Errorf("CommitRange = {Start:%d, Count:%d}, want {Start:4, Count:0}",
			block.CommitRange.Start, block.CommitRange.Count)
	}
	if block.ActualRange.Start != 4 || block.ActualRange.Count != 3 {
		t.Errorf("ActualRange = {Start:%d, Count:%d}, want {Start:4, Count:3}",
			block.ActualRange.Start, block.ActualRange.Count)
	}
}

func TestExtractChangeBlocks_DeletedLines(t *testing.T) {
	// Test case: 2 lines deleted at line 5
	fileDiff := &FileDiff{
		NewPath:  "test.go",
		OldLines: []string{"line1", "line2", "line3", "line4", "deleted1", "deleted2", "line5"},
		NewLines: []string{"line1", "line2", "line3", "line4", "line5"},
		Hunks: []*DiffHunk{
			{
				Header: HunkHeader{
					OldRange: HeaderRange{Start: 4, Count: 4},
					NewRange: HeaderRange{Start: 4, Count: 2},
				},
				Lines: []DiffLine{
					{Type: DiffContext, Content: "line4"},
					{Type: DiffRemoved, Content: "deleted1"},
					{Type: DiffRemoved, Content: "deleted2"},
					{Type: DiffContext, Content: "line5"},
				},
			},
		},
	}

	fc := extractChangeBlocks(fileDiff)

	if len(fc.ChangeBlocks) != 1 {
		t.Fatalf("expected 1 change block, got %d", len(fc.ChangeBlocks))
	}

	block := fc.ChangeBlocks[0]
	if block.Type != LinesDeleted {
		t.Errorf("Type = %v, want LinesDeleted", block.Type)
	}
	if block.CommitRange.Start != 5 || block.CommitRange.Count != 2 {
		t.Errorf("CommitRange = {Start:%d, Count:%d}, want {Start:5, Count:2}",
			block.CommitRange.Start, block.CommitRange.Count)
	}
	if block.ActualRange.Start != 5 || block.ActualRange.Count != 0 {
		t.Errorf("ActualRange = {Start:%d, Count:%d}, want {Start:5, Count:0}",
			block.ActualRange.Start, block.ActualRange.Count)
	}
}

func TestExtractChangeBlocks_MixedChanges(t *testing.T) {
	// Test case: deletion at line 5, addition at line 10
	fileDiff := &FileDiff{
		NewPath: "test.go",
		OldLines: []string{
			"line1", "line2", "line3", "line4", "deleted1",
			"line5", "line6", "line7", "line8", "line9", "line10",
		},
		NewLines: []string{
			"line1", "line2", "line3", "line4",
			"line5", "line6", "line7", "line8", "line9", "added1", "line10",
		},
		Hunks: []*DiffHunk{
			{
				Header: HunkHeader{
					OldRange: HeaderRange{Start: 4, Count: 3},
					NewRange: HeaderRange{Start: 4, Count: 2},
				},
				Lines: []DiffLine{
					{Type: DiffContext, Content: "line4"},
					{Type: DiffRemoved, Content: "deleted1"},
					{Type: DiffContext, Content: "line5"},
				},
			},
			{
				Header: HunkHeader{
					OldRange: HeaderRange{Start: 9, Count: 2},
					NewRange: HeaderRange{Start: 9, Count: 3},
				},
				Lines: []DiffLine{
					{Type: DiffContext, Content: "line9"},
					{Type: DiffAdded, Content: "added1"},
					{Type: DiffContext, Content: "line10"},
				},
			},
		},
	}

	fc := extractChangeBlocks(fileDiff)

	if len(fc.ChangeBlocks) != 2 {
		t.Fatalf("expected 2 change blocks, got %d", len(fc.ChangeBlocks))
	}

	// First block: deletion
	block1 := fc.ChangeBlocks[0]
	if block1.Type != LinesDeleted {
		t.Errorf("block1.Type = %v, want LinesDeleted", block1.Type)
	}
	if block1.CommitRange.Start != 5 || block1.CommitRange.Count != 1 {
		t.Errorf("block1.CommitRange = {Start:%d, Count:%d}, want {Start:5, Count:1}",
			block1.CommitRange.Start, block1.CommitRange.Count)
	}

	// Second block: addition
	block2 := fc.ChangeBlocks[1]
	if block2.Type != LinesAdded {
		t.Errorf("block2.Type = %v, want LinesAdded", block2.Type)
	}
	if block2.ActualRange.Start != 10 || block2.ActualRange.Count != 1 {
		t.Errorf("block2.ActualRange = {Start:%d, Count:%d}, want {Start:10, Count:1}",
			block2.ActualRange.Start, block2.ActualRange.Count)
	}
}

func TestBlockChangeType_String(t *testing.T) {
	tests := []struct {
		typ  BlockChangeType
		want string
	}{
		{LinesAdded, "added"},
		{LinesDeleted, "deleted"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.typ.String()
			if got != tt.want {
				t.Errorf("BlockChangeType.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractChangeBlocks_EmptyFile(t *testing.T) {
	// Test case: Empty file (no hunks, no lines)
	fileDiff := &FileDiff{
		NewPath:  "empty.go",
		OldLines: []string{},
		NewLines: []string{},
		Hunks:    []*DiffHunk{},
	}

	fc := extractChangeBlocks(fileDiff)

	if len(fc.ChangeBlocks) != 0 {
		t.Fatalf("expected 0 change blocks, got %d", len(fc.ChangeBlocks))
	}
	if len(fc.CommitLines) != 0 {
		t.Fatalf("expected 0 commit lines, got %d", len(fc.CommitLines))
	}
	if len(fc.ActualLines) != 0 {
		t.Fatalf("expected 0 actual lines, got %d", len(fc.ActualLines))
	}
}
