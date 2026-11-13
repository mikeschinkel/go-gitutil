//go:build !aix && !darwin && !dragonfly && !freebsd && !illumos && !linux && !netbsd && !openbsd && !solaris

package gitutil

import (
	"fmt"

	"github.com/mikeschinkel/go-dt"
)

func ensureSameFilesystem(_, _ dt.DirPath) error {
	return fmt.Errorf("cache hardlink requirement is not supported on this platform")
}
