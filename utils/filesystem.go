package utils

import (
	"os"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

// ExpandPath Expands a file path
// 1. replace tilde with users home dir
// 2. expands embedded environment variables
// 3. cleans the path, e.g. /a/b/../c -> /a/c
// Note, it has limitations, e.g. ~someuser/tmp will not be expanded
func ExpandPath(p string) string {
	if i := strings.Index(p, ":"); i > 0 {
		return p
	}

	if i := strings.Index(p, "@"); i > 0 {
		return p
	}

	if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~\\") {
		if home := homeDir(); home != "" {
			p = home + p[1:]
		}
	}

	return path.Clean(os.ExpandEnv(p))
}

func FileExist(filePath string) bool {
	if _, err := os.Stat(filePath); err != nil && os.IsNotExist(err) {
		return false
	}

	return true
}

func DirExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return true, err
}

func RemoveDir(path string) error {
	return os.RemoveAll(path)
}

func ProjectRootDir() string {
	_, b, _, _ := runtime.Caller(0)

	dir := filepath.Dir(filepath.Dir(b))
	return dir
}

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}
