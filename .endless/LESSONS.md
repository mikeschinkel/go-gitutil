# Lessons Learned

A capture log of corrections, newest last. Entries are appended by
`endless lesson write`, which commits this file on the main checkout as it
writes — so recording a lesson never waits on a task landing, and never
dirties a worktree.

## Format

```
### [YYYY-MM-DD] One-line summary
<the lesson — what went wrong, why, and the rule that replaces it>
- **Project**: <project>
```

---

## Entries

### [2026-11-12] A brainstorm outcome is the current best understanding, not dogma; say so in it and expect it to evolve during implementation
In E-2548 Mike asked for a note that the outcome is what we best understand and may need to evolve during implementation, because he finds that once an outcome is written it gets treated as settled and he has to fight to change a design when implementation reveals a need. Rule: write that caveat into every brainstorm or research outcome, and when implementation finds a reason to deviate, treat the outcome as a starting point to revise, not an authority to defend.
- **Project**: go-gitutil
