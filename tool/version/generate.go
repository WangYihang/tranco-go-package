//go:build ignore
// +build ignore

package main

import (
	"log/slog"
	"os"
	"path/filepath"
	"text/template"

	"github.com/WangYihang/tranco-go-package/pkg/version"
)

func main() {
	v, err := version.GetVersionFromGit()
	if err != nil {
		slog.Error("error occured while parsing version from git", slog.String("error", err.Error()))
	}

	versionFilepath := filepath.Join("pkg", "version", "default.go")
	fd, err := os.Create(versionFilepath)
	if err != nil {
		slog.Error("error occured while creating version file", slog.String("file", versionFilepath), slog.String("error", err.Error()))
	}
	defer fd.Close()

	tmpl, err := template.New("").Parse(`package version

var (
	Tag string = "{{ . }}"
	// Version is the current version of the program
	Version string = "0.0.1"
	// CommitHash is the current commit hash of the program
	CommitHash string = "unknown"
	// BuildTime is the current build time of the program
	BuildTime string = "unknown"
)
`)
	if err != nil {
		slog.Error("error occured while parsing version template file", slog.String("error", err.Error()))
	}

	err = tmpl.Execute(fd, v)
	if err != nil {
		slog.Error("error occured while rendering version template file", slog.String("version", v), slog.String("error", err.Error()))
	}
}
