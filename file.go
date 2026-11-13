package gitutil

import (
	"github.com/mikeschinkel/go-dt"
)

type File struct {
	Filepath dt.RelFilepath
	Status   FileStatus
}

func NewFile(filepath dt.RelFilepath) File {
	return File{Filepath: filepath}
}
