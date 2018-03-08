package utils

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	AppRoot string
)

func init() {
	AppRoot = GetAppRoot()
}

func GetAppRoot() string {
	file, err := exec.LookPath(os.Args[0])
	if err != nil {
		return ""
	}
	p, err := filepath.Abs(file)
	if err != nil {
		return ""
	}
	return filepath.Dir(p)
}

func MergePath(args ...string) string {
	for i, e := range args {
		if e != "" {
			return filepath.Join(AppRoot, filepath.Clean(strings.Join(args[i:], string(filepath.Separator))))
		}
	}
	return AppRoot
}

func PathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

func MakePath(path string) error {
	err := os.MkdirAll(path, os.ModePerm)
	return err
}
