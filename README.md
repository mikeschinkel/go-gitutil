# go-gitutil

go-gitutil runs git for Go programs: creating, opening and inspecting
repositories, staging and committing, tags, diffs and worktrees. It shells out
to the `git` binary rather than reimplementing git, so it behaves as the
command line does.

## Install

```sh
go get github.com/mikeschinkel/go-gitutil
```

go-gitutil is provisional as of v0.1.0: its API may change in minor releases.

## Usage

A `Repo` is built with `NewRepo`, which runs no git. Methods that need only the
directory work at once; methods that read the current branch or its upstream
need `Open` first, and each method's doc comment says which it is. `Init`,
`Clone` and `Open` exist both as `Repo` methods and as package-level functions
that return a `Repo`.

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-gitutil"
)

func main() {
	ctx := context.Background()

	// NewRepo runs no git; methods that need only the directory work at once.
	repo := gitutil.NewRepo("/path/to/repo", &gitutil.RepoArgs{
		StdIO: dt.NewDefaultStdIO(), // show commit, clone and push output
	})
	head, err := repo.RevParse(ctx, "HEAD")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("HEAD is", head)

	// Open finds the repository's top level and reads its branch and upstream.
	opened, err := gitutil.Open(ctx, "/path/to/repo", nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("on branch", opened.Branch, "tracking", opened.Remote)
}
```

## RepoArgs

Every git process runs through one function and inherits its `Repo`'s
`RepoArgs`:

- `StdIO` receives output meant for a person, such as commit, clone and push.
  Nil captures it instead. Output go-gitutil parses is always captured.
- `Env` adds `KEY=VALUE` entries to git's environment.
- `GitPath` names the git binary. Empty finds `git` on `PATH`.
- `Strict` runs git isolated from the machine.

An operation's `Args` type embeds `RepoArgs`, so one call can add to the
`Repo`'s settings for that call only. A call can turn `Strict` on but never off.

### Strict mode

`Strict` keeps git away from the machine's configuration, credentials and
network. Each git process gets an environment built from nothing:

- `HOME` set to an empty directory go-gitutil creates and removes
- `GIT_CONFIG_GLOBAL=/dev/null` and `GIT_CONFIG_NOSYSTEM=1`, so no global or
  system config applies
- `GIT_ALLOW_PROTOCOL=file`, so only local remotes work
- `GIT_TERMINAL_PROMPT=0`, so git never prompts for credentials
- `-c user.useConfigOnly=true`, so git never guesses an identity

then `Env`. `PATH` is not passed, so Strict mode requires `GitPath` and fails
before running git without it. Tests use it to get the same git behavior on
every machine.

## Development

`test/` is a separate module that holds the tests against the exported API.
CI builds, vets, lints and tests both modules with `GOWORK=off`, against the
versions each `go.mod` requires.

## Releasing

Releases are made by the **Release** GitHub Actions workflow, which tests the
module, then tags and publishes it. Tags are never created by hand.

## License

MIT; see [LICENSE](LICENSE).
