package version

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// PV is the current version object of the program
var PV ProgramVersion

func init() {
	PV.Tag = Tag
	PV.Version = Version
	PV.CommitHash = CommitHash
	PV.BuildTime = BuildTime
}

// ProgramVersion is the version object of the program
type ProgramVersion struct {
	Tag        string `json:"tag"`
	Version    string `json:"version"`
	CommitHash string `json:"commit_hash"`
	BuildTime  string `json:"build_time"`
}

// Short returns the short version of the program
func (v ProgramVersion) Short() string {
	return v.Tag
}

// GetVersionFromGit derives a "<latest-tag>-<short-commit>" version string
// from the local git repository by shelling out to the git CLI. This is used
// by `go generate` (to refresh pkg/version/default.go before a release) and
// by TestVersion; it intentionally avoids depending on a full git
// implementation (with its SSH/crypto transitive dependencies) just to read
// a tag and a commit hash.
func GetVersionFromGit() (string, error) {
	tag, err := runGit("describe", "--tags", "--abbrev=0")
	if err != nil {
		return "", fmt.Errorf("no tags found: %w", err)
	}

	commitHash, err := runGit("rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to resolve HEAD commit: %w", err)
	}
	if len(commitHash) < 8 {
		return "", fmt.Errorf("unexpected commit hash %q", commitHash)
	}

	return fmt.Sprintf("%s-%s", tag, commitHash[:8]), nil
}

func runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(stdout.String()), nil
}
