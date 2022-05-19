package app

import (
	"os"
	"path"
	"path/filepath"
	"runtime"
	"time"

	"go.uber.org/zap"
)

const (
	ServiceName = "prolog-service"
)

type Dependencies struct {
	ServiceName     string
	Build           string
	Host            string
	DebugHost       string
	ReadTimout      time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	Kubernetes      KubeInfo
	Shutdown        chan os.Signal
	Logger          *zap.SugaredLogger
}

type KubeInfo struct {
	Pod       string
	PodIP     string
	Node      string
	Namespace string
}

// RootDir is designed to return the absolute path of the directory of
// this project
func RootDir() string {
	_, b, _, _ := runtime.Caller(0)
	d := path.Join(path.Dir(b))
	return filepath.Dir(d)
}
