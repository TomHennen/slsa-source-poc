package store

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
)

func fixPredType(predType string) string {
	return strings.ReplaceAll(predType, "/", "_")
}

func computePath(commit, predType string, dataDigest []byte) string {
	// predicate types are often URIs, so replace the /.
	fixedPred := fixPredType(predType)
	return path.Join(commit, fixedPred, fmt.Sprintf("%x.json", dataDigest))
}

func Store(commit, predType, repoPath, dataPath string) (string, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", err
	}
	worktree, err := repo.Worktree()
	if err != nil {
		return "", err
	}
	status, err := worktree.Status()
	if err != nil {
		return "", err
	}
	if !status.IsClean() {
		return "", errors.New("Repo must be clean to store metadata")
	}
	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: "refs/slsa/commits",
	})
	if err != nil {
		return "", err
	}

	// Write the data
	data, err := os.ReadFile(dataPath)
	if err != nil {
		return "", err
	}

	dataDigest := sha256.Sum256(data)

	storePath := computePath(commit, predType, dataDigest[:])

	// Create the entire path if it doesn't already exist
	if err := os.MkdirAll(filepath.Dir(storePath), 0770); err != nil {
		return "", err
	}

	err = os.WriteFile(storePath, data, 0644)
	if err != nil {
		return "", err
	}

	return storePath, nil
}
