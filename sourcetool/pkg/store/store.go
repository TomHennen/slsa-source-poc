package store

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path"
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

func Store(commit, predType, dataPath string) (string, error) {
	data, err := os.ReadFile(dataPath)
	if err != nil {
		return "", err
	}

	dataDigest := sha256.Sum256(data)

	storePath := computePath(commit, predType, dataDigest[:])

	repo, err := git.PlainOpen(".")
	if err != nil {
		return "", err
	}
	worktree, err := repo.Worktree()
	if err != nil {
		return "", err
	}
	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: "refs/slsa/commits",
	})
	if err != nil {
		return "", err
	}

	return storePath, nil
}
