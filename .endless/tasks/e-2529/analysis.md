# Decided with Mike (2026-11-16, in E-2624's session)
- Q1 README: A. Write a short README before tagging. All four sibling repos checked (go-dt, go-diffutils, go-fsfix, go-cliutil) have one.
- Q2 LICENSE history: A. Merge GitHub's LICENSE-only commit with --allow-unrelated-histories. Force-pushing would overwrite a published commit on a public repo. Rebasing would change every local commit hash, which the Endless ledger and open task branches refer to.
- Q3 E-2530 before the tag: A. Tag first. LatestTag takes *LatestTagArgs, so E-2530 can add a field later without breaking callers.

# Decided without asking
- The first version proposed is v0.1.0, as the sibling repos started.
- PLAN-CONDENSED-DIFF-TYPES.md is removed. It proposes moving the condensed-diff types into gitutil. Those types and BuildCondensedDiff read only old/new lines and change ranges, which diffutils.DiffContent already carries, so gitutil is not their home. Where they belong is a separate question put to Mike.
- E-2602 is not needed before the release: the root diff_parser_test.go imports only testing.
- The analysis's earlier steps (commit the go.mod change, require published versions, drop replace, Go 1.27, .gitignore) were done by E-2600 and E-2624, so the plan checks them instead of redoing them.
- gomion is built from a scratch go.work because no installed or checked-out gomion builds with `repo setup` today. E-2540 tracks the bug that makes `repo setup --check` exit non-zero.
