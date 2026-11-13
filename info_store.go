package gitutils

import (
	"context"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"io/fs"
	"os"
	"strings"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/dtx"
)

type InfoStore struct {
	RepoPath dt.DirPath
	Filename dt.RelFilepath
	fs       fs.FS
}

func NewInfoStore(repoPath dt.DirPath, filename dt.RelFilepath) *InfoStore {
	return &InfoStore{
		RepoPath: repoPath,
		Filename: filename,
	}
}

func (s *InfoStore) Save(data []byte) (err error) {
	var file *os.File
	var fp dt.Filepath

	fp, err = s.GetFilepath()
	if err != nil {
		goto end
	}

	file, err = fp.Create()
	if err != nil {
		err = NewErr(
			dt.ErrFailedtoCreateFile,
			fp.ErrKV(),
			err,
		)
		goto end
	}
	defer dt.CloseOrLog(file)

	_, err = file.Write(data)
	if err != nil {
		err = NewErr(
			dt.ErrFailedToWriteToFile,
			fp.ErrKV(),
			err,
		)
		goto end
	}

end:
	if err != nil {
		err = WithErr(err,
			dt.ErrFailedToSaveFile,
			ErrFailedToSaveGitInfoFile,
		)
	}
	return err
}

func (s *InfoStore) SaveJSON(data any) (err error) {
	var jsonData []byte

	// Use JSON v2 with pretty printing via jsontext.WithIndent
	jsonData, err = jsonv2.Marshal(data, jsontext.WithIndent("  "))
	if err != nil {
		err = NewErr(dt.ErrFailedToMarshalJSON, ErrFailedToSaveJSONFile, err)
		goto end
	}

	err = s.Save(jsonData)

end:
	return err
}

func (s *InfoStore) Load() (data []byte, err error) {
	var fSys fs.FS
	var fp dt.Filepath

	fSys, err = s.getFS()
	if err != nil {
		err = WithErr(ErrFailedToGetFileSystem, err)
		goto end
	}

	data, err = s.Filename.ReadFile(fSys)
	if dtx.NoFileOrDirErr(err) {
		err = NewErr(
			dt.ErrFileNotExist,
			fp.ErrKV(),
			err,
		)
		goto end
	}
	if err != nil {
		goto end
	}

end:
	if err != nil {
		err = NewErr(
			dt.ErrFailedToLoadFile,
			ErrFailedToLoadGitInfoFile,
			err,
		)
	}
	return data, err
}

func (s *InfoStore) LoadJSON(data any, opts ...jsonv2.Options) (err error) {
	var jsonData []byte
	jsonData, err = s.Load()
	if err != nil {
		goto end
	}

	// Use JSON v2 with any provided options (including custom unmarshalers)
	err = jsonv2.Unmarshal(jsonData, data, opts...)
	if err != nil {
		err = NewErr(
			dt.ErrFailedToUnmarshalJSON,
			ErrFailedToLoadGitInfoFile,
			err,
		)
		goto end
	}

end:
	if err != nil {
		err = WithErr(err, ErrFailedToLoadJSONFile)
	}
	return err
}

func (s *InfoStore) GetDirPath() (dp dt.DirPath, err error) {
	var exists bool

	dp, err = InfoDir(context.Background(), s.RepoPath)
	if err != nil {
		goto end
	}
	exists, err = dp.Exists()
	switch {
	case !exists:
		err = NewErr(
			dt.ErrDirNotExist,
			ErrGitInfoPathNotExist,
			dp.ErrKV(),
		)
		goto end
	case err != nil:
		err = NewErr(
			dt.ErrFileSystem,
			ErrFailedInspectingGitInfoPath,
			dp.ErrKV(),
			err,
		)
		goto end
	}
end:
	return dp, err
}

func (s *InfoStore) GetFilepath() (fp dt.Filepath, err error) {
	var dp dt.DirPath

	dp, err = s.GetDirPath()
	if err != nil {
		err = NewErr(ErrFailedToGetGitInfoFilepath, err)
		goto end
	}
	fp = dt.FilepathJoin(dp, s.Filename)
end:
	return fp, err
}

func (s *InfoStore) getFS() (_ fs.FS, err error) {
	var dir dt.DirPath

	if s.fs != nil {
		goto end
	}

	dir, err = s.GetDirPath()
	if err != nil {
		goto end
	}

	s.fs = dt.DirFS(dir)

end:
	return s.fs, err
}

func (s *InfoStore) ErrKV() ErrKV {
	fp, err := s.GetFilepath()
	if err != nil {
		fp = ""
	}
	return fp.ErrKV()
}

// InfoDir returns the directory git keeps repoRoot's info/ files in, as git
// itself resolves it. In a plain clone that is <repoRoot>/.git/info. In a
// worktree or a submodule .git is a file pointing elsewhere, and info/ lives
// in the git directory it points to, so the path cannot be built by joining
// ".git/info" onto repoRoot.
func InfoDir(ctx context.Context, repoRoot dt.DirPath) (dp dt.DirPath, err error) {
	var out string

	out, err = newGitRunner(repoRoot).runGit(ctx, runGitArgs{
		Command: []string{"rev-parse", "--path-format=absolute", "--git-path", "info"},
	})
	if err != nil {
		err = NewErr(ErrFailedInspectingGitInfoPath, "repo_root", repoRoot, err)
		goto end
	}
	dp = dt.DirPath(strings.TrimSpace(out))

end:
	return dp, err
}
