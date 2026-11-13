package gitutil

import (
	"github.com/mikeschinkel/go-dt"
)

// FileFilter is a function that returns true if file should be included
type FileFilter func(dt.RelFilepath) bool
