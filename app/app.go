package app

import (
	"path"
	"path/filepath"
	"runtime"
)

const (
	ServiceName = "prolog-service"
)

// RootDir is designed to return the absolute path of the directory of
// this project
func RootDir() string {
	_, b, _, _ := runtime.Caller(0)
	d := path.Join(path.Dir(b))
	return filepath.Dir(d)
}
