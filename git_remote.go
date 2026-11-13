package gitutil

import (
	"fmt"

	"github.com/mikeschinkel/go-dt"
)

// GitRemote is a remote and a branch on it, such as the upstream a local
// branch tracks.
type GitRemote struct {
	Name   dt.GitRemoteName
	Branch dt.GitBranchName
}

func (r GitRemote) String() string {
	return fmt.Sprintf("%s/%s", r.Name, r.Branch)
}

func (r GitRemote) IsValid() bool {
	return len(r.Name) > 0 && len(r.Branch) > 0
}
