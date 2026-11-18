package mock

import (
	"context"

	"league.dev/merge-gatekeeper/internal/github"
)

type Client struct {
	GetCombinedStatusFunc          func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error)
	ListCheckRunsForRefFunc        func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error)
	ListRepositoryWorkflowRunsFunc func(ctx context.Context, owner, repo string, opts *github.ListWorkflowRunsOptions) (*github.WorkflowRuns, *github.Response, error)
}

func (c *Client) GetCombinedStatus(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
	return c.GetCombinedStatusFunc(ctx, owner, repo, ref, opts)
}

func (c *Client) ListCheckRunsForRef(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
	return c.ListCheckRunsForRefFunc(ctx, owner, repo, ref, opts)
}

func (c *Client) ListRepositoryWorkflowRuns(ctx context.Context, owner, repo string, opts *github.ListWorkflowRunsOptions) (*github.WorkflowRuns, *github.Response, error) {
	return c.ListRepositoryWorkflowRunsFunc(ctx, owner, repo, opts)
}

var (
	_ github.Client = &Client{}
)
