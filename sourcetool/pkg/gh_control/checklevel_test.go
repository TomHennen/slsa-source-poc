package gh_control
func TestRuleMeetsRequiresReview(t *testing.T) {
	// Test data.
	rules := &github.PullRequestBranchRule{
		RulesetID: 3,
		Parameters: github.PullRequestBranchRuleParameters{
			RequiredApprovingReviewCount: 1,
			DismissStaleReviewsOnPush:    true,
			RequireCodeOwnerReview:       true,
			RequireLastPushApproval:      true,
		},
	}

	// Call the function.
	ghc := &GitHubConnection{
		Client: github.NewClient(nil),
	}

	// Call the function.
	result := ghc.ruleMeetsRequiresReview(rules)
	if !result {
		t.Fatalf("ruleMeetsRequiresReview failed: %v", rules)
	}
}

func TestRuleNotMeetsRequiresReview(t *testing.T) {
	// Test data.
	rules := &github.PullRequestBranchRule{
		RulesetID: 3,
		Parameters: github.PullRequestBranchRuleParameters{
			RequiredApprovingReviewCount: 0,
			DismissStaleReviewsOnPush:    false,
			RequireCodeOwnerReview:       false,
			RequireLastPushApproval:      false,
		},
	}

	// Call the function.
	ghc := &GitHubConnection{
		Client: github.NewClient(nil),
	}

	// Call the function.
	result := ghc.ruleMeetsRequiresReview(rules)
	if result {
		t.Fatalf("ruleMeetsRequiresReview failed: %v", rules)
	}
}
import (
	"context"
	"testing"
	"time"

	"github.com/google/go-github/v69/github"
	"github.com/slsa-framework/slsa-source-poc/sourcetool/pkg/slsa_types"
)

type MockGitHubClient struct {
	// Add fields for mocking the GitHub API response.
	Client                  *github.Client
	GetRulesForBranchFunc   func(ctx context.Context, owner, repo, branch string) (*github.BranchRules, *github.Response, error)
	CommitActivityFunc      func(ctx context.Context, commit string) (*activity, error)
	GetRulesetFunc          func(ctx context.Context, owner, repo string, rulesetID int64, includeAll bool) (*github.RepositoryRuleset, *github.Response, error)
	GetPullRequestFunc      func(ctx context.Context, owner, repo string, rulesetID int64, includeAll bool) ([]*github.PullRequestBranchRule, *github.Response, error)
}

func (m *MockGitHubClient) GetRulesForBranch(ctx context.Context, owner, repo, branch string) (*github.BranchRules, *github.Response, error) {
	if m.GetRulesForBranchFunc != nil {
		return m.GetRulesForBranchFunc(ctx, owner, repo, branch)
	}
	return nil, nil
}

func (m *MockGitHubClient) CommitActivity(ctx context.Context, commit string) (*activity, error) {
	if m.CommitActivityFunc != nil {
		return m.CommitActivityFunc(ctx, commit)
	}
	return nil, nil
}

func (m *MockGitHubClient) GetRuleset(ctx context.Context, owner, repo string, rulesetID int64, includeAll bool) (*github.RepositoryRuleset, *github.Response, error) {
	if m.GetRulesetFunc != nil {
		return m.GetRulesetFunc(ctx, owner, repo, rulesetID, includeAll)
	}
	return nil, nil
}

func (m *MockGitHubClient) GetPullRequest(ctx context.Context, owner, repo string, rulesetID int64, includeAll bool) ([]*github.PullRequestBranchRule, *github.Response, error) {
	if m.GetPullRequestFunc != nil {
		return m.GetPullRequestFunc(ctx, owner, repo, rulesetID, includeAll)
	}
	return nil, nil
}

func TestComputeReviewControl(t *testing.T) {
	// Create a mock client for testing.
	mockClient := &MockGitHubClient{
		Client: github.NewClient(nil),
	}

	// Expected data.
	expectedReviewControl := &slsa_types.Control{Name: slsa_types.ReviewEnforced, Since: time.Now()}

	// Create mock pull request rules.
	mockPullRequestRules := []*github.PullRequestBranchRule{
		{
			RulesetID: 3,
			Parameters: github.PullRequestBranchRuleParameters{
				RequiredApprovingReviewCount: 1,
				DismissStaleReviewsOnPush:    true,
				RequireCodeOwnerReview:       true,
				RequireLastPushApproval:      true,
			},
		},
	}

	// Call the function.
	ghc := &GitHubConnection{
		Client: mockClient,
		Owner:  "testOwner",
		Repo:   "testRepo",
		Branch: "main",
	}

	//Mock GetPullRequest
	mockClient.GetPullRequestFunc = func(ctx context.Context, owner, repo string, rulesetID int64, includeAll bool) ([]*github.PullRequestBranchRule, *github.Response, error) {
		return mockPullRequestRules, nil, nil
	}
	//Mock GetRuleset
	mockClient.GetRulesetFunc = func(ctx context.Context, owner, repo string, rulesetID int64, includeAll bool) (*github.RepositoryRuleset, *github.Response, error) {
		return &github.RepositoryRuleset{
			Enforcement: "active",
			UpdatedAt: &github.Timestamp{Time: time.Now()},
		}, nil, nil
	}

	// Call the function.
	control, err := ghc.computeReviewControl(context.Background(), mockPullRequestRules)
	if err != nil {
		t.Fatalf("computeReviewControl failed: %v", err)
	}

	// Check if the correct controls are present.
	if control.Name != expectedReviewControl.Name {
		t.Errorf("Control Name mismatch: got %s, expected %s", control.Name, expectedReviewControl.Name)
	}
	if control.Since.IsZero() {
		t.Errorf("Since is not defined")
	}
	if control.Since.Unix() != time.Now().Unix() {
		t.Errorf("Since mismatch: got %v, expected %v", control.Since.Unix(), time.Now().Unix())
	}
func TestComputeReviewControlWithNoRules(t *testing.T) {
	// Create a mock client for testing.
	mockClient := &MockGitHubClient{
		Client: github.NewClient(nil),
	}

	// Expected data.
	expectedReviewControl := &slsa_types.Control{Name: slsa_types.ReviewEnforced, Since: time.Now()}

	// Create mock pull request rules.
	mockPullRequestRules := []*github.PullRequestBranchRule{}
	
	// Call the function.
	ghc := &GitHubConnection{
		Client: mockClient,
		Owner:  "testOwner",
		Repo:   "testRepo",
		Branch: "main",
	}

	//Mock GetPullRequest
	mockClient.GetPullRequestFunc = func(ctx context.Context, owner, repo string, rulesetID int64, includeAll bool) ([]*github.PullRequestBranchRule, *github.Response, error) {
		return mockPullRequestRules, nil, nil
	}
	//Mock GetRuleset
	mockClient.GetRulesetFunc = func(ctx context.Context, owner, repo string, rulesetID int64, includeAll bool) (*github.RepositoryRuleset, *github.Response, error) {
		return &github.RepositoryRuleset{
			Enforcement: "active",
			UpdatedAt: &github.Timestamp{Time: time.Now()},
		}, nil, nil
	}

	// Call the function.
	control, err := ghc.computeReviewControl(context.Background(), mockPullRequestRules)
	if err == nil {
		t.Fatalf("computeReviewControl failed: %v", err)
	}
	if control != nil {
		t.Errorf("Control mismatch: got %s, expected %s", control, nil)
	}
}
