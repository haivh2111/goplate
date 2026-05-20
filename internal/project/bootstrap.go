package project

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// InitGit runs `git init` + `git add .` + `git commit -m <message>` in dir.
// Returns nil on success, or a wrapped error showing the failed command's output.
func InitGit(dir, commitMessage string) error {
	steps := [][]string{
		{"git", "init", "-q"},
		{"git", "add", "."},
		{"git", "commit", "-q", "-m", commitMessage},
	}
	for _, argv := range steps {
		cmd := exec.Command(argv[0], argv[1:]...)
		cmd.Dir = dir
		// Provide a deterministic identity in case git is unconfigured —
		// only at the env level so we don't touch the user's global config.
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=goplate",
			"GIT_AUTHOR_EMAIL=goplate@example.com",
			"GIT_COMMITTER_NAME=goplate",
			"GIT_COMMITTER_EMAIL=goplate@example.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git %v failed: %w\n%s", argv[1:], err, out)
		}
	}
	return nil
}

// CopyEnv copies srcRel to dstRel inside root. Both paths are relative to root.
// Returns nil if srcRel doesn't exist (skip without error).
func CopyEnv(root, srcRel, dstRel string) error {
	srcAbs := filepath.Join(root, srcRel)
	dstAbs := filepath.Join(root, dstRel)
	src, err := os.Open(srcAbs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer src.Close()
	dst, err := os.Create(dstAbs)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	return err
}

// GoModTidy runs `go mod tidy` in dir.
func GoModTidy(dir string) error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go mod tidy failed: %w\n%s", err, out)
	}
	return nil
}
