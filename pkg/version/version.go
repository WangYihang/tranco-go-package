package version

import (
	"fmt"
	"log"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// PV is the current version object of the program
var PV ProgramVersion

func init() {
	PV.Tag, _ = GetVersionFromGit()
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

func GetVersionFromGit() (string, error) {
	r, err := git.PlainOpen(".")
	if err != nil {
		log.Printf("Error opening git repository: %v", err)
		return "", err
	}

	tagRefs, err := r.Tags()
	if err != nil {
		log.Printf("Error getting tags: %v", err)
		return "", err
	}

	var latestTag *plumbing.Reference
	err = tagRefs.ForEach(func(t *plumbing.Reference) error {
		latestTag = t
		return nil
	})
	if err != nil {
		log.Printf("Error iterating tags: %v", err)
		return "", err
	}

	var tag string
	if latestTag == nil {
		return "", fmt.Errorf("no tags found")
	}
	tag = latestTag.Name().Short()

	head, err := r.Head()
	if err != nil {
		log.Printf("Error getting HEAD: %v", err)
		return "", err
	}
	commitHash := head.Hash().String()

	fullVersion := fmt.Sprintf("%s-%s", tag, commitHash[:8])
	return fullVersion, nil
}
