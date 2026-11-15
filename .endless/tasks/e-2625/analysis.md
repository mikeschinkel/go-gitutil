Shape agreed in the E-2608 planning chat:

```go
// Ignored runs one git check-ignore over paths and returns the subset git ignores.
func (r *Repo) Ignored(ctx context.Context, paths []dt.RelDirPath) ([]dt.RelDirPath, error)

// gomion composes it with goutil:
nested, err := goutil.FindNestedModules(dir, nil)
ignored, err := repo.Ignored(ctx, nested)
nested = slices.DeleteFunc(nested, func(d dt.RelDirPath) bool { return slices.Contains(ignored, d) })
```

Use `check-ignore --stdin -z` with paths NUL-separated on stdin through the one run function (E-2600 gives it stdin), and read the NUL-separated answer. check-ignore exits 1 when no path is ignored, which is not an error here. Whether the parameter is []dt.RelDirPath or a path union covering files too is for the implementer to settle with Mike; the known caller passes directories. Tests in test/ (E-2602).
