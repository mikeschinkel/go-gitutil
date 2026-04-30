# Plan: Extract Condensed Diff Types to gitutils

## Goal

Move the format-agnostic condensed diff data types and builder function from `gomtui/context_info_pane_model.go` into `gitutils`. These types have no TUI dependency and belong with the diff infrastructure.

## Background

The condensed diff feature creates a compact summary of file changes: a few context lines, changed lines (capped at 5), and truncation indicators. Currently the types and builder live in `gomtui` but they operate purely on `gitutils.FileChanges` data with no lipgloss/Bubble Tea dependency.

After this extraction, `gomtui` will import these types from `gitutils`, and the new `teadiff` package (in go-tealeaves) will also consume them without depending on gomion.

## Types to Move

From `gomtui/context_info_pane_model.go`:

### `CondensedBlock` (currently lines 51-58)

```go
type CondensedBlock struct {
    Type          string   // "added" or "deleted"
    LineCount     int
    ContextBefore []string
    ChangedLines  []string
    ContextAfter  []string
    IsTruncated   bool
}
```

### `FileCondensedDiff` (currently lines 60-64, with new FileStatus field)

```go
type FileStatus int

const (
    FileModified FileStatus = iota
    FileNew
    FileDeleted
)

type FileCondensedDiff struct {
    Path   dt.RelFilepath
    Status FileStatus
    Blocks []CondensedBlock
}
```

### `BuildCondensedDiff()` function (currently lines ~800-850)

Converts `*FileChanges` into `FileCondensedDiff`. Pure data transformation, no TUI imports.

## Changes

### 1. New file: `gitutils/condensed_diff.go`

Create this file containing:
- `FileStatus` type and constants (`FileModified`, `FileNew`, `FileDeleted`)
- `CondensedBlock` struct
- `FileCondensedDiff` struct
- `BuildCondensedDiff(changes *FileChanges) FileCondensedDiff` function

The builder logic is a direct move from `gomtui/context_info_pane_model.go`. No behavioral changes.

### 2. Update `gomtui/context_info_pane_model.go`

- Remove `FileStatus`, `CondensedBlock`, `FileCondensedDiff` type declarations
- Remove `BuildCondensedDiff()` function
- Replace all references with `gitutils.FileStatus`, `gitutils.CondensedBlock`, `gitutils.FileCondensedDiff`, `gitutils.BuildCondensedDiff()`
- Update `FileNew` / `FileDeleted` / `FileModified` references to `gitutils.FileNew` etc.

Affected locations:
- `DirectoryContext.Files` field type: `[]FileCondensedDiff` → `[]gitutils.FileCondensedDiff`
- `viewDirectory()` method: references to `FileNew`, `FileDeleted`, `CondensedBlock`
- `BuildCondensedDiff()` call sites in `loadDirectoryContext()`
- `SetDirectoryContext()` line count estimation loop

### 3. Update `gomtui/batch_assignment_model.go`

- `loadDirectoryContext()` call to `BuildCondensedDiff()` → `gitutils.BuildCondensedDiff()`

## Verification

1. `go build -o /dev/null ./gommod/gitutils/...`
2. `go build -o /dev/null ./gommod/gomtui/...`
3. `go test ./gommod/gitutils/...`
4. `go test ./gommod/gomtui/...`
5. Run `gomion commit` and verify directory context pane renders identically

## Dependencies

- None. This is a pure refactor moving existing code.
- Must complete BEFORE the teadiff package (Layer 2) can consume these types.
- The gomtui integration plan (Layer 3) depends on both this and teadiff.
