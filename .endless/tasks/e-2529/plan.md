# Plan: first tagged release of go-gitutil

Already true on main, checked rather than redone: root go.mod requires go-diffutils v0.1.0, go-dt v0.7.2 and go-dt/dtx v0.3.0 (each its latest tag) with no replace; every go.mod says go 1.27.0; .gitignore lists go.work and go.work.sum.

## Steps
1. Write README.md: what go-gitutil does, `go get`, a short example of NewRepo, Open and RepoArgs, and what Strict mode isolates. Draw it from doc.go and RepoArgs' doc comment.
2. Remove PLAN-CONDENSED-DIFF-TYPES.md.
3. Run `git merge --allow-unrelated-histories origin/main` to bring in GitHub's LICENSE commit. No force push, no rebase.
4. Generate .github/workflows/test.yml and release.yml with `gomion repo setup <repo>`. The installed gomion lacks `repo setup` and gomion's main does not build in its own workspace (teamodal v0.1.0 requires teautils v0.0.0), so build cmd/gomion with a scratch go.work, outside both repos, that adds go-tealeaves/teautils. Don't edit gomion, and don't hand-edit the generated files. Because of E-2627, `--check` exits non-zero on its own arguments, so read its go-gitutil line rather than its exit code.
5. Run GOWORK=off build, vet and test on the root and test/ modules, and golangci-lint v2.15.0 on both.
6. Push main to origin. Create no tag.
7. Watch the first Test run on the pushed commit; it must pass with no warning or notice annotations.
8. Record in the task's notes a proposed version, v0.1.0, and a release description, for the owner to run the Release workflow with.
