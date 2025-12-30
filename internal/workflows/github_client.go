package workflows

import (
	"context"
	"fmt"

	"github.com/fyrsmithlabs/contextd/internal/config"
	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

// GitHubClientConfig is defined in types.go for shared use across workflows

// NewGitHubClient creates a GitHub client with proper authentication.
func NewGitHubClient(ctx context.Context, token config.Secret) (*github.Client, error) {
	if !token.IsSet() {
		return nil, fmt.Errorf("GitHub token not set")
	}

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token.Value()})
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc), nil
}
