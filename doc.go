// Package gitutil runs git for Go programs: creating, opening and inspecting
// repositories, staging and committing, tags, diffs and worktrees.
//
// A Repo is built with NewRepo, which runs no git, and is opened with
// (*Repo).Open when a method needs the current branch or its upstream. Each
// method's doc comment says whether it needs Open. Init, Clone and Open exist
// both as Repo methods and as package-level functions that return a Repo.
//
// Every git process runs through one function and inherits the Repo's
// RepoArgs: where human-readable output goes (StdIO), extra environment
// (Env), which git binary to run (GitPath), and whether git runs isolated
// from the machine's configuration and credentials (Strict).
package gitutil
