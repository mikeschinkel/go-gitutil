# Proposed release

Run the **Release** workflow on go-gitutil's main once the first Test run on the pushed main passes:

- module: `root`
- version: `v0.1.0`
- description:

> First tagged release of go-gitutil, the git helpers extracted from gomion. It runs the git binary to create, clone, open and inspect repositories, stage and commit, read and create tags, parse diffs, and manage worktrees. Every git process runs with the Repo's RepoArgs (StdIO, Env, GitPath, and Strict, which isolates git from the machine's configuration and credentials). It also classifies commits by conventional-commit type into commit signals. It requires go-diffutils v0.1.0, go-dt v0.7.2 and go-dt/dtx v0.3.0. The API is provisional and may change in minor releases. LatestTag does not yet find directory-prefixed tags of nested modules (E-2530).
