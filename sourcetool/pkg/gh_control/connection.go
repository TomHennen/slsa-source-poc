package gh_control

import (
	"context"
	"fmt"

	"github.com/google/go-github/v69/github"
)

type GitHubConnection struct {
	Client              *github.Client
	Owner, Repo, Branch string
}

func NewGhConnection(owner, repo, branch string) *GitHubConnection {
	return &GitHubConnection{
		Client: github.NewClient(nil),
		Owner:  owner,
		Repo:   repo,
		Branch: branch}
}

// Uses the provide token for auth.
// If the token is the empty string this is a no-op.
func (ghc *GitHubConnection) WithAuthToken(token string) *GitHubConnection {
	if token != "" {
		ghc.Client = ghc.Client.WithAuthToken(token)
	}
	return ghc
}

// Returns the fully qualified branch (e.g. 'refs/heads/main').
func (ghc *GitHubConnection) GetFullBranch() string {
	return fmt.Sprintf("refs/heads/%s", ghc.Branch)
}

// Returns the URI of the repo this connection tracks.
func (ghc *GitHubConnection) GetRepoUri() string {
	return fmt.Sprintf("https://github.com/%s/%s", ghc.Owner, ghc.Repo)
}

func (ghc *GitHubConnection) GetLatestCommit(ctx context.Context) (string, error) {
	branch, _, err := ghc.Client.Repositories.GetBranch(ctx, ghc.Owner, ghc.Repo, ghc.Branch, 1)
	if err != nil {
		return "", fmt.Errorf("could not get info on specified branch %s: %w", ghc.Branch, err)
	}
	return *branch.Commit.SHA, nil
}
