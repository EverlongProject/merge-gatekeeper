package github

import (
	"context"

	"github.com/google/go-github/v69/github"
	"golang.org/x/oauth2"
)

type (
	ListOptions    = github.ListOptions
	CombinedStatus = github.CombinedStatus
	RepoStatus     = github.RepoStatus
	Response       = github.Response
)

type (
	CheckRun                = github.CheckRun
	ListCheckRunsOptions    = github.ListCheckRunsOptions
	ListCheckRunsResults    = github.ListCheckRunsResults
	WorkflowRun             = github.WorkflowRun
	WorkflowRuns            = github.WorkflowRuns
	ListWorkflowRunsOptions = github.ListWorkflowRunsOptions
)

type Client interface {
	GetCombinedStatus(ctx context.Context, owner, repo, ref string, opts *ListOptions) (*CombinedStatus, *Response, error)
	ListCheckRunsForRef(ctx context.Context, owner, repo, ref string, opts *ListCheckRunsOptions) (*ListCheckRunsResults, *Response, error)
	ListRepositoryWorkflowRuns(ctx context.Context, owner, repo string, opts *ListWorkflowRunsOptions) (*WorkflowRuns, *Response, error)
}

type client struct {
	ghc *github.Client
}

func NewClient(ctx context.Context, token string) Client {
	return &client{
		ghc: github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(
			&oauth2.Token{
				AccessToken: token,
			},
		))),
	}
}

func (c *client) GetCombinedStatus(ctx context.Context, owner, repo, ref string, opts *ListOptions) (*CombinedStatus, *Response, error) {
	return c.ghc.Repositories.GetCombinedStatus(ctx, owner, repo, ref, opts)
}

func (c *client) ListCheckRunsForRef(ctx context.Context, owner, repo, ref string, opts *ListCheckRunsOptions) (*ListCheckRunsResults, *Response, error) {
	return c.ghc.Checks.ListCheckRunsForRef(ctx, owner, repo, ref, opts)
}

func (c *client) ListRepositoryWorkflowRuns(ctx context.Context, owner, repo string, opts *ListWorkflowRunsOptions) (*WorkflowRuns, *Response, error) {
	return c.ghc.Actions.ListRepositoryWorkflowRuns(ctx, owner, repo, opts)
}
