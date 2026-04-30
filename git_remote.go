package gitutils

import (
	"fmt"
)

type RemoteName string
type GitRemote struct {
	Name   RemoteName
	Branch GitRef
}

func (r GitRemote) String() string {
	return fmt.Sprintf("%s/%s", r.Name, r.Branch)
}

func (r GitRemote) IsValid() bool {
	return len(r.Name) > 0 && len(r.Branch) > 0
}
