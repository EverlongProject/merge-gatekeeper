package status

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	ghapi "github.com/google/go-github/v69/github"
	"league.dev/merge-gatekeeper/internal/github"
	"league.dev/merge-gatekeeper/internal/github/mock"
	"league.dev/merge-gatekeeper/internal/validators"
)

func stringPtr(str string) *string {
	return &str
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestCreateValidator(t *testing.T) {
	tests := map[string]struct {
		c                 github.Client
		opts              []Option
		want              validators.Validator
		wantErr           bool
		wantErrSubstrings []string
	}{
		"returns Validator when option is not empty": {
			c: &mock.Client{},
			opts: []Option{
				WithGitHubOwnerAndRepo("test-owner", "test-repo"),
				WithGitHubRef("sha"),
				WithSelfJob("job"),
				WithIgnoredJobs("job-01,job-02"),
				WithIgnoreDynamicGitHubWorkflows(true),
			},
			want: &statusValidator{
				client:                       &mock.Client{},
				owner:                        "test-owner",
				repo:                         "test-repo",
				ref:                          "sha",
				selfJobName:                  "job",
				ignoredJobs:                  []string{"job-01", "job-02"},
				ignoreDynamicGitHubWorkflows: true,
			},
			wantErr: false,
		},
		"returns Validator when there are duplicate options": {
			c: &mock.Client{},
			opts: []Option{
				WithGitHubOwnerAndRepo("test", "test-repo"),
				WithGitHubRef("sha"),
				WithGitHubRef("sha-01"),
				WithSelfJob("job"),
				WithSelfJob("job-01"),
			},
			want: &statusValidator{
				client:                       &mock.Client{},
				owner:                        "test",
				repo:                         "test-repo",
				ref:                          "sha-01",
				selfJobName:                  "job-01",
				ignoreDynamicGitHubWorkflows: true,
			},
			wantErr: false,
		},
		"returns Validator when invalid string is provided for ignored jobs": {
			c: &mock.Client{},
			opts: []Option{
				WithGitHubOwnerAndRepo("test", "test-repo"),
				WithGitHubRef("sha"),
				WithGitHubRef("sha-01"),
				WithSelfJob("job"),
				WithSelfJob("job-01"),
				WithIgnoredJobs(","),
			},
			want:    nil,
			wantErr: true,
		},
		"returns error when ignored jobs contains an empty item": {
			c: &mock.Client{},
			opts: []Option{
				WithGitHubOwnerAndRepo("test", "test-repo"),
				WithGitHubRef("sha"),
				WithSelfJob("job-01"),
				WithIgnoredJobs("job-01,,job-03"),
			},
			want:    nil,
			wantErr: true,
		},
		"returns error listing all empty ignored job entries": {
			c: &mock.Client{},
			opts: []Option{
				WithGitHubOwnerAndRepo("test", "test-repo"),
				WithGitHubRef("sha"),
				WithSelfJob("job-01"),
				WithIgnoredJobs(",job-01,,job-03,"),
			},
			want:              nil,
			wantErr:           true,
			wantErrSubstrings: []string{"ignored jobs contains empty entry at position 1", "ignored jobs contains empty entry at position 3", "ignored jobs contains empty entry at position 5"},
		},
		"returns Validator when later ignored jobs option overrides earlier invalid one": {
			c: &mock.Client{},
			opts: []Option{
				WithGitHubOwnerAndRepo("test-owner", "test-repo"),
				WithGitHubRef("sha"),
				WithSelfJob("job"),
				WithIgnoredJobs(","),
				WithIgnoredJobs("job-03,job-04"),
			},
			want: &statusValidator{
				client:                       &mock.Client{},
				owner:                        "test-owner",
				repo:                         "test-repo",
				ref:                          "sha",
				selfJobName:                  "job",
				ignoredJobs:                  []string{"job-03", "job-04"},
				ignoreDynamicGitHubWorkflows: true,
			},
			wantErr: false,
		},
		"returns error when option is empty": {
			c:       &mock.Client{},
			want:    nil,
			wantErr: true,
		},
		"returns error when client is nil": {
			c: nil,
			opts: []Option{
				WithGitHubOwnerAndRepo("test", "test-repo"),
				WithGitHubRef("sha"),
				WithGitHubRef("sha-01"),
				WithSelfJob("job"),
				WithSelfJob("job-01"),
			},
			want:    nil,
			wantErr: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := CreateValidator(tt.c, tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateValidator error = %v, wantErr: %v", err, tt.wantErr)
				return
			}
			for _, expected := range tt.wantErrSubstrings {
				if err == nil || !strings.Contains(err.Error(), expected) {
					t.Fatalf("CreateValidator() error = %v, want substring %q", err, expected)
				}
			}
			if tt.want != nil {
				gotValidator, ok := got.(*statusValidator)
				if !ok {
					t.Fatalf("CreateValidator() type = %T, want *statusValidator", got)
				}
				wantValidator := tt.want.(*statusValidator)
				if gotValidator.owner != wantValidator.owner ||
					gotValidator.repo != wantValidator.repo ||
					gotValidator.ref != wantValidator.ref ||
					gotValidator.selfJobName != wantValidator.selfJobName ||
					!reflect.DeepEqual(gotValidator.ignoredJobs, wantValidator.ignoredJobs) ||
					gotValidator.ignoreDynamicGitHubWorkflows != wantValidator.ignoreDynamicGitHubWorkflows ||
					len(gotValidator.optionErrs) != 0 ||
					len(gotValidator.ignoredJobsErrs) != 0 ||
					gotValidator.client != tt.c {
					t.Errorf("CreateValidator() = %v, want %v", gotValidator, wantValidator)
				}
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateValidator() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestName(t *testing.T) {
	tests := map[string]struct {
		c    github.Client
		opts []Option
		want string
	}{
		"Name returns the correct job name which gets overridden": {
			c: &mock.Client{},
			opts: []Option{
				WithGitHubOwnerAndRepo("test-owner", "test-repo"),
				WithGitHubRef("sha"),
				WithSelfJob("job"),
				WithIgnoredJobs("job-01,job-02"),
			},
			want: "job",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := CreateValidator(tt.c, tt.opts...)
			if err != nil {
				t.Errorf("Unexpected error with CreateValidator: %v", err)
				return
			}
			if tt.want != got.Name() {
				t.Errorf("Job name didn't match, want: %s, got: %v", tt.want, got.Name())
			}
		})
	}
}

func Test_statusValidator_Validate(t *testing.T) {
	type test struct {
		selfJobName                  string
		ignoredJobs                  []string
		ignoreDynamicGitHubWorkflows bool
		client                       github.Client
		ctx                          context.Context
		wantErr                      bool
		wantErrStr                   string
		wantStatus                   validators.Status
	}
	tests := map[string]test{
		"returns error when listGhaStatuses return an error": {
			client: &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return nil, nil, errors.New("err")
				},
			},
			wantErr:    true,
			wantStatus: nil,
			wantErrStr: "err",
		},
		"returns succeeded status and nil when there is no job": {
			client: &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					return &github.ListCheckRunsResults{}, nil, nil
				},
			},
			wantErr: false,
			wantStatus: &status{
				succeeded:    true,
				totalJobs:    []string{},
				completeJobs: []string{},
				ignoredJobs:  []string{},
				errJobs:      []string{},
			},
		},
		"returns succeeded status and nil when there is one job, which is itself": {
			selfJobName: "self-job",
			client: &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{
						Statuses: []*github.RepoStatus{
							{
								Context: stringPtr("self-job"),
								State:   stringPtr(pendingState), // should be irrelevant
							},
						},
					}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					return &github.ListCheckRunsResults{}, nil, nil
				},
			},
			wantErr: false,
			wantStatus: &status{
				succeeded:    true,
				totalJobs:    []string{},
				completeJobs: []string{},
				ignoredJobs:  []string{},
				errJobs:      []string{},
			},
		},
		"returns failed status and nil when there is one job": {
			client: &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{
						Statuses: []*github.RepoStatus{
							{
								Context: stringPtr("job"),
								State:   stringPtr(pendingState),
							},
						},
					}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					return &github.ListCheckRunsResults{}, nil, nil
				},
			},
			wantErr: false,
			wantStatus: &status{
				succeeded:    false,
				totalJobs:    []string{"job"},
				completeJobs: []string{},
				ignoredJobs:  []string{},
				errJobs:      []string{},
			},
		},
		"returns error when there is a failed job": {
			selfJobName: "self-job",
			client: &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{
						Statuses: []*github.RepoStatus{
							{
								Context: stringPtr("job-01"),
								State:   stringPtr(successState),
							},
							{
								Context: stringPtr("job-02"),
								State:   stringPtr(errorState),
							},
							{
								Context: stringPtr("self-job"),
								State:   stringPtr(pendingState),
							},
						},
					}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					return &github.ListCheckRunsResults{}, nil, nil
				},
			},
			wantErr: true,
			wantErrStr: (&status{
				totalJobs: []string{
					"job-01", "job-02",
				},
				completeJobs: []string{
					"job-01",
				},
				errJobs: []string{
					"job-02",
				},
				ignoredJobs: []string{},
			}).Detail(),
		},
		"returns error when there is a failed job with failure state": {
			selfJobName: "self-job",
			client: &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{
						Statuses: []*github.RepoStatus{
							{
								Context: stringPtr("job-01"),
								State:   stringPtr(successState),
							},
							{
								Context: stringPtr("job-02"),
								State:   stringPtr(failureState),
							},
							{
								Context: stringPtr("self-job"),
								State:   stringPtr(pendingState),
							},
						},
					}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					return &github.ListCheckRunsResults{}, nil, nil
				},
			},
			wantErr: true,
			wantErrStr: (&status{
				totalJobs: []string{
					"job-01", "job-02",
				},
				completeJobs: []string{
					"job-01",
				},
				errJobs: []string{
					"job-02",
				},
				ignoredJobs: []string{},
			}).Detail(),
		},
		"returns failed status and nil when successful job count is less than total": {
			selfJobName: "self-job",
			client: &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{
						Statuses: []*github.RepoStatus{
							{
								Context: stringPtr("job-01"),
								State:   stringPtr(successState),
							},
							{
								Context: stringPtr("job-02"),
								State:   stringPtr(pendingState),
							},
							{
								Context: stringPtr("self-job"),
								State:   stringPtr(pendingState),
							},
						},
					}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					return &github.ListCheckRunsResults{}, nil, nil
				},
			},
			wantErr: false,
			wantStatus: &status{
				succeeded: false,
				totalJobs: []string{
					"job-01",
					"job-02",
				},
				completeJobs: []string{
					"job-01",
				},
				errJobs:     []string{},
				ignoredJobs: []string{},
			},
		},
		"returns succeeded status and nil when validation is success": {
			selfJobName: "self-job",
			client: &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{
						Statuses: []*github.RepoStatus{
							{
								Context: stringPtr("job-01"),
								State:   stringPtr(successState),
							},
							{
								Context: stringPtr("job-02"),
								State:   stringPtr(successState),
							},
							{
								Context: stringPtr("self-job"),
								State:   stringPtr(pendingState),
							},
						},
					}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					return &github.ListCheckRunsResults{}, nil, nil
				},
			},
			wantErr: false,
			wantStatus: &status{
				succeeded: true,
				totalJobs: []string{
					"job-01",
					"job-02",
				},
				completeJobs: []string{
					"job-01",
					"job-02",
				},
				errJobs:     []string{},
				ignoredJobs: []string{},
			},
		},
		"returns succeeded status and nil when only an ignored job is failing": {
			selfJobName: "self-job",
			ignoredJobs: []string{"job-02", "job-03"}, // String input here should be already TrimSpace'd
			client: &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{
						Statuses: []*github.RepoStatus{
							{
								Context: stringPtr("job-01"),
								State:   stringPtr(successState),
							},
							{
								Context: stringPtr("job-02"),
								State:   stringPtr(errorState),
							},
							{
								Context: stringPtr("self-job"),
								State:   stringPtr(pendingState),
							},
						},
					}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					return &github.ListCheckRunsResults{}, nil, nil
				},
			},
			wantErr: false,
			wantStatus: &status{
				succeeded:    true,
				totalJobs:    []string{"job-01"},
				completeJobs: []string{"job-01"},
				errJobs:      []string{},
				ignoredJobs:  []string{"job-02", "job-03"},
			},
		},
		"returns succeeded status and nil when only an ignored job is failing, with failure state": {
			selfJobName: "self-job",
			ignoredJobs: []string{"job-02", "job-03"},
			client: &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{
						Statuses: []*github.RepoStatus{
							{
								Context: stringPtr("job-01"),
								State:   stringPtr(successState),
							},
							{
								Context: stringPtr("job-02"),
								State:   stringPtr(failureState),
							},
							{
								Context: stringPtr("self-job"),
								State:   stringPtr(pendingState),
							},
						},
					}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					return &github.ListCheckRunsResults{}, nil, nil
				},
			},
			wantErr: false,
			wantStatus: &status{
				succeeded:    true,
				totalJobs:    []string{"job-01"},
				completeJobs: []string{"job-01"},
				errJobs:      []string{},
				ignoredJobs:  []string{"job-02", "job-03"},
			},
		},
		"returns succeeded status when only dynamic workflow checks fail and ignoring is enabled": {
			ignoreDynamicGitHubWorkflows: true,
			client: &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					return &github.ListCheckRunsResults{
						CheckRuns: []*github.CheckRun{
							{
								Name:       stringPtr("Agent"),
								Status:     stringPtr(checkRunCompletedStatus),
								Conclusion: stringPtr("failure"),
								DetailsURL: stringPtr("https://github.com/test-owner/test-repo/actions/runs/1/job/2"),
								CheckSuite: &ghapi.CheckSuite{ID: ghapi.Int64(123)},
							},
							{
								Name:       stringPtr("Cleanup artifacts"),
								Status:     stringPtr(checkRunCompletedStatus),
								Conclusion: stringPtr("failure"),
								DetailsURL: stringPtr("https://github.com/test-owner/test-repo/actions/runs/1/job/3"),
								CheckSuite: &ghapi.CheckSuite{ID: ghapi.Int64(123)},
							},
						},
					}, nil, nil
				},
				ListRepositoryWorkflowRunsFunc: func(ctx context.Context, owner, repo string, opts *github.ListWorkflowRunsOptions) (*github.WorkflowRuns, *github.Response, error) {
					return &github.WorkflowRuns{
						WorkflowRuns: []*github.WorkflowRun{
							{
								Name: stringPtr("Copilot code review"),
								Path: stringPtr("dynamic/copilot-pull-request-reviewer/copilot-pull-request-reviewer"),
							},
						},
					}, nil, nil
				},
			},
			wantErr: false,
			wantStatus: &status{
				succeeded:    true,
				totalJobs:    []string{},
				completeJobs: []string{},
				ignoredJobs:  []string{},
				errJobs:      []string{},
				ignoredDynamicWorkflows: []ignoredDynamicWorkflowGroup{
					{
						WorkflowName: "Copilot code review",
						WorkflowPath: "dynamic/copilot-pull-request-reviewer/copilot-pull-request-reviewer",
						Jobs:         []string{"Agent", "Cleanup artifacts"},
					},
				},
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			sv := &statusValidator{
				owner:                        "test-owner",
				repo:                         "test-repo",
				selfJobName:                  tt.selfJobName,
				ignoredJobs:                  tt.ignoredJobs,
				ignoreDynamicGitHubWorkflows: tt.ignoreDynamicGitHubWorkflows,
				client:                       tt.client,
			}
			got, err := sv.Validate(tt.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("statusValidator.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if err.Error() != tt.wantErrStr {
					t.Errorf("statusValidator.Validate() error.Error() = %s, wantErrStr %s", err.Error(), tt.wantErrStr)
				}
			}
			if !reflect.DeepEqual(got, tt.wantStatus) {
				t.Errorf("statusValidator.Validate() status = %v, want %v", got, tt.wantStatus)
			}
		})
	}
}

func Test_statusValidator_listStatuses(t *testing.T) {
	type fields struct {
		repo        string
		owner       string
		ref         string
		selfJobName string
		client      github.Client
	}
	type test struct {
		fields                       fields
		ignoreDynamicGitHubWorkflows bool
		ctx                          context.Context
		wantErr                      bool
		want                         []*ghaStatus
		workflowLookupCalls          *int
		wantWorkflowLookupCalls      int
	}
	tests := map[string]test{
		"succeeds to get job statuses even if the same job exists": func() test {
			c := &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{
						Statuses: []*github.RepoStatus{
							// The first element here is the latest state.
							{
								Context: stringPtr("job-01"),
								State:   stringPtr(successState),
							},
							{
								Context: stringPtr("job-01"), // Same as above job name, and thus should be disregarded as old job status.
								State:   stringPtr(errorState),
							},
						},
					}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					return &github.ListCheckRunsResults{
						CheckRuns: []*github.CheckRun{
							// The first element here is the latest state.
							{
								Name:   stringPtr("job-02"),
								Status: stringPtr("failure"),
							},
							{
								Name:       stringPtr("job-02"), // Same as above job name, and thus should be disregarded as old job status.
								Status:     stringPtr(checkRunCompletedStatus),
								Conclusion: stringPtr(checkRunNeutralConclusion),
							},
							{
								Name:       stringPtr("job-03"),
								Status:     stringPtr(checkRunCompletedStatus),
								Conclusion: stringPtr(checkRunNeutralConclusion),
							},
							{
								Name:       stringPtr("job-04"),
								Status:     stringPtr(checkRunCompletedStatus),
								Conclusion: stringPtr(checkRunSuccessConclusion),
							},
							{
								Name:       stringPtr("job-05"),
								Status:     stringPtr(checkRunCompletedStatus),
								Conclusion: stringPtr("failure"),
							},
							{
								Name:       stringPtr("job-06"),
								Status:     stringPtr(checkRunCompletedStatus),
								Conclusion: stringPtr(checkRunSkipConclusion),
							},
						},
					}, nil, nil
				},
			}
			return test{
				fields: fields{
					client:      c,
					selfJobName: "self-job",
					owner:       "test-owner",
					repo:        "test-repo",
					ref:         "main",
				},
				wantErr: false,
				want: []*ghaStatus{
					{
						Job:   "job-01",
						State: successState,
					},
					{
						Job:   "job-02",
						State: pendingState,
					},
					{
						Job:   "job-03",
						State: successState,
					},
					{
						Job:   "job-04",
						State: successState,
					},
					{
						Job:   "job-05",
						State: errorState,
					},
				},
			}
		}(),
		"returns error when the GetCombinedStatus returns an error": func() test {
			c := &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return nil, nil, errors.New("err")
				},
			}
			return test{
				fields: fields{
					client:      c,
					selfJobName: "self-job",
					owner:       "test-owner",
					repo:        "test-repo",
					ref:         "main",
				},
				wantErr: true,
			}
		}(),
		"returns error when the GetCombinedStatus response is invalid": func() test {
			c := &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{
						Statuses: []*github.RepoStatus{
							{},
						},
					}, nil, nil
				},
			}
			return test{
				fields: fields{
					client:      c,
					selfJobName: "self-job",
					owner:       "test-owner",
					repo:        "test-repo",
					ref:         "main",
				},
				wantErr: true,
			}
		}(),
		"returns error when the ListCheckRunsForRef returns an error": func() test {
			c := &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					return nil, nil, errors.New("error")
				},
			}
			return test{
				fields: fields{
					client:      c,
					selfJobName: "self-job",
					owner:       "test-owner",
					repo:        "test-repo",
					ref:         "main",
				},
				wantErr: true,
			}
		}(),
		"returns error when the ListCheckRunsForRef response is invalid": func() test {
			c := &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					return &github.ListCheckRunsResults{
						CheckRuns: []*github.CheckRun{
							{},
						},
					}, nil, nil
				},
			}
			return test{
				fields: fields{
					client:      c,
					selfJobName: "self-job",
					owner:       "test-owner",
					repo:        "test-repo",
					ref:         "main",
				},
				wantErr: true,
			}
		}(),
		"returns error when a completed check run has no conclusion": func() test {
			c := &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					return &github.ListCheckRunsResults{
						CheckRuns: []*github.CheckRun{
							{
								Name:   stringPtr("job-01"),
								Status: stringPtr(checkRunCompletedStatus),
							},
						},
					}, nil, nil
				},
			}
			return test{
				fields: fields{
					client:      c,
					selfJobName: "self-job",
					owner:       "test-owner",
					repo:        "test-repo",
					ref:         "main",
				},
				wantErr: true,
			}
		}(),
		"returns nil when no error occurs": func() test {
			c := &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{
						Statuses: []*github.RepoStatus{
							{
								Context: stringPtr("job-01"),
								State:   stringPtr(successState),
							},
						},
					}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					return &github.ListCheckRunsResults{
						CheckRuns: []*github.CheckRun{
							{
								Name:   stringPtr("job-02"),
								Status: stringPtr("failure"),
							},
							{
								Name:       stringPtr("job-03"),
								Status:     stringPtr(checkRunCompletedStatus),
								Conclusion: stringPtr(checkRunNeutralConclusion),
							},
							{
								Name:       stringPtr("job-04"),
								Status:     stringPtr(checkRunCompletedStatus),
								Conclusion: stringPtr(checkRunSuccessConclusion),
							},
							{
								Name:       stringPtr("job-05"),
								Status:     stringPtr(checkRunCompletedStatus),
								Conclusion: stringPtr("failure"),
							},
							{
								Name:       stringPtr("job-06"),
								Status:     stringPtr(checkRunCompletedStatus),
								Conclusion: stringPtr(checkRunSkipConclusion),
							},
						},
					}, nil, nil
				},
			}
			return test{
				fields: fields{
					client:      c,
					selfJobName: "self-job",
					owner:       "test-owner",
					repo:        "test-repo",
					ref:         "main",
				},
				wantErr: false,
				want: []*ghaStatus{
					{
						Job:   "job-01",
						State: successState,
					},
					{
						Job:   "job-02",
						State: pendingState,
					},
					{
						Job:   "job-03",
						State: successState,
					},
					{
						Job:   "job-04",
						State: successState,
					},
					{
						Job:   "job-05",
						State: errorState,
					},
				},
			}
		}(),
		"marks dynamic workflow check runs for ignoring and reuses the workflow lookup cache": func() test {
			lookupCalls := 0
			c := &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					return &github.ListCheckRunsResults{
						CheckRuns: []*github.CheckRun{
							{
								Name:       stringPtr("Agent"),
								Status:     stringPtr(checkRunCompletedStatus),
								Conclusion: stringPtr("failure"),
								DetailsURL: stringPtr("https://github.com/test-owner/test-repo/actions/runs/1/job/2"),
								CheckSuite: &ghapi.CheckSuite{ID: ghapi.Int64(123)},
							},
							{
								Name:       stringPtr("Cleanup artifacts"),
								Status:     stringPtr(checkRunCompletedStatus),
								Conclusion: stringPtr("failure"),
								DetailsURL: stringPtr("https://github.com/test-owner/test-repo/actions/runs/1/job/3"),
								CheckSuite: &ghapi.CheckSuite{ID: ghapi.Int64(123)},
							},
						},
					}, nil, nil
				},
				ListRepositoryWorkflowRunsFunc: func(ctx context.Context, owner, repo string, opts *github.ListWorkflowRunsOptions) (*github.WorkflowRuns, *github.Response, error) {
					lookupCalls++
					return &github.WorkflowRuns{
						WorkflowRuns: []*github.WorkflowRun{
							{
								Name: stringPtr("Copilot code review"),
								Path: stringPtr("dynamic/copilot-pull-request-reviewer/copilot-pull-request-reviewer"),
							},
						},
					}, nil, nil
				},
			}
			want := []*ghaStatus{
				{
					Job:                        "Agent",
					State:                      errorState,
					IgnoredDynamicWorkflowName: "Copilot code review",
					IgnoredDynamicWorkflowPath: "dynamic/copilot-pull-request-reviewer/copilot-pull-request-reviewer",
				},
				{
					Job:                        "Cleanup artifacts",
					State:                      errorState,
					IgnoredDynamicWorkflowName: "Copilot code review",
					IgnoredDynamicWorkflowPath: "dynamic/copilot-pull-request-reviewer/copilot-pull-request-reviewer",
				},
			}
			return test{
				fields: fields{
					client:      c,
					selfJobName: "self-job",
					owner:       "test-owner",
					repo:        "test-repo",
					ref:         "main",
				},
				ignoreDynamicGitHubWorkflows: true,
				wantErr:                      false,
				want:                         want,
				workflowLookupCalls:          &lookupCalls,
				wantWorkflowLookupCalls:      1,
			}
		}(),
		"records failed workflow lookups while keeping the check run in scope": func() test {
			c := &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					return &github.ListCheckRunsResults{
						CheckRuns: []*github.CheckRun{
							{
								Name:       stringPtr("Analyze (go)"),
								Status:     stringPtr(checkRunCompletedStatus),
								Conclusion: stringPtr(checkRunSuccessConclusion),
								DetailsURL: stringPtr("https://github.com/test-owner/test-repo/actions/runs/9/job/2"),
								CheckSuite: &ghapi.CheckSuite{ID: ghapi.Int64(456)},
							},
						},
					}, nil, nil
				},
				ListRepositoryWorkflowRunsFunc: func(ctx context.Context, owner, repo string, opts *github.ListWorkflowRunsOptions) (*github.WorkflowRuns, *github.Response, error) {
					return nil, nil, errors.New("lookup failed")
				},
			}
			return test{
				fields: fields{
					client:      c,
					selfJobName: "self-job",
					owner:       "test-owner",
					repo:        "test-repo",
					ref:         "main",
				},
				ignoreDynamicGitHubWorkflows: true,
				wantErr:                      false,
				want: []*ghaStatus{
					{
						Job:                 "Analyze (go)",
						State:               successState,
						FailedLookupCommand: "gh api --method GET repos/test-owner/test-repo/actions/runs -F check_suite_id=456",
					},
				},
			}
		}(),
		"does not ignore check runs from repository workflows": func() test {
			lookupCalls := 0
			c := &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					return &github.ListCheckRunsResults{
						CheckRuns: []*github.CheckRun{
							{
								Name:       stringPtr("test (ubuntu-latest, go1.23)"),
								Status:     stringPtr(checkRunCompletedStatus),
								Conclusion: stringPtr(checkRunSuccessConclusion),
								DetailsURL: stringPtr("https://github.com/test-owner/test-repo/actions/runs/11/job/22"),
								CheckSuite: &ghapi.CheckSuite{ID: ghapi.Int64(789)},
							},
						},
					}, nil, nil
				},
				ListRepositoryWorkflowRunsFunc: func(ctx context.Context, owner, repo string, opts *github.ListWorkflowRunsOptions) (*github.WorkflowRuns, *github.Response, error) {
					lookupCalls++
					return &github.WorkflowRuns{
						WorkflowRuns: []*github.WorkflowRun{
							{
								Name: stringPtr("CI"),
								Path: stringPtr(".github/workflows/ci.yml"),
							},
						},
					}, nil, nil
				},
			}
			return test{
				fields: fields{
					client:      c,
					selfJobName: "self-job",
					owner:       "test-owner",
					repo:        "test-repo",
					ref:         "main",
				},
				ignoreDynamicGitHubWorkflows: true,
				wantErr:                      false,
				want: []*ghaStatus{
					{
						Job:   "test (ubuntu-latest, go1.23)",
						State: successState,
					},
				},
				workflowLookupCalls:     &lookupCalls,
				wantWorkflowLookupCalls: 1,
			}
		}(),
		"does not attempt workflow lookup for github advanced security checks": func() test {
			lookupCalls := 0
			c := &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					return &github.ListCheckRunsResults{
						CheckRuns: []*github.CheckRun{
							{
								Name:       stringPtr("CodeQL"),
								Status:     stringPtr(checkRunCompletedStatus),
								Conclusion: stringPtr(checkRunSuccessConclusion),
								DetailsURL: stringPtr("https://github.com/test-owner/test-repo/actions/runs/19/job/21"),
								CheckSuite: &ghapi.CheckSuite{ID: ghapi.Int64(456)},
								App:        &ghapi.App{Slug: ghapi.String(githubAdvancedSecurityAppSlug)},
							},
						},
					}, nil, nil
				},
				ListRepositoryWorkflowRunsFunc: func(ctx context.Context, owner, repo string, opts *github.ListWorkflowRunsOptions) (*github.WorkflowRuns, *github.Response, error) {
					lookupCalls++
					return &github.WorkflowRuns{}, nil, nil
				},
			}
			return test{
				fields: fields{
					client:      c,
					selfJobName: "self-job",
					owner:       "test-owner",
					repo:        "test-repo",
					ref:         "main",
				},
				ignoreDynamicGitHubWorkflows: true,
				wantErr:                      false,
				want: []*ghaStatus{
					{
						Job:   "CodeQL",
						State: successState,
					},
				},
				workflowLookupCalls:     &lookupCalls,
				wantWorkflowLookupCalls: 0,
			}
		}(),
		"caches empty workflow run lookups within a single poll": func() test {
			lookupCalls := 0
			c := &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					return &github.ListCheckRunsResults{
						CheckRuns: []*github.CheckRun{
							{
								Name:       stringPtr("CodeQL / Analyze (go)"),
								Status:     stringPtr(checkRunCompletedStatus),
								Conclusion: stringPtr(checkRunSuccessConclusion),
								DetailsURL: stringPtr("https://github.com/test-owner/test-repo/actions/runs/19/job/21"),
								CheckSuite: &ghapi.CheckSuite{ID: ghapi.Int64(456)},
							},
							{
								Name:       stringPtr("CodeQL / Analyze (javascript)"),
								Status:     stringPtr(checkRunCompletedStatus),
								Conclusion: stringPtr(checkRunSuccessConclusion),
								DetailsURL: stringPtr("https://github.com/test-owner/test-repo/actions/runs/19/job/22"),
								CheckSuite: &ghapi.CheckSuite{ID: ghapi.Int64(456)},
							},
						},
					}, nil, nil
				},
				ListRepositoryWorkflowRunsFunc: func(ctx context.Context, owner, repo string, opts *github.ListWorkflowRunsOptions) (*github.WorkflowRuns, *github.Response, error) {
					lookupCalls++
					return &github.WorkflowRuns{}, nil, nil
				},
			}
			return test{
				fields: fields{
					client:      c,
					selfJobName: "self-job",
					owner:       "test-owner",
					repo:        "test-repo",
					ref:         "main",
				},
				ignoreDynamicGitHubWorkflows: true,
				wantErr:                      false,
				want: []*ghaStatus{
					{
						Job:                 "CodeQL / Analyze (go)",
						State:               successState,
						FailedLookupCommand: "gh api --method GET repos/test-owner/test-repo/actions/runs -F check_suite_id=456",
					},
					{
						Job:                 "CodeQL / Analyze (javascript)",
						State:               successState,
						FailedLookupCommand: "gh api --method GET repos/test-owner/test-repo/actions/runs -F check_suite_id=456",
					},
				},
				workflowLookupCalls:     &lookupCalls,
				wantWorkflowLookupCalls: 1,
			}
		}(),
		"succeeds to retrieve 100 statuses": func() test {
			num_statuses := 100
			statuses := make([]*github.RepoStatus, num_statuses)
			checkRuns := make([]*github.CheckRun, num_statuses)
			expectedGhaStatuses := make([]*ghaStatus, num_statuses)
			for i := 0; i < num_statuses; i++ {
				statuses[i] = &github.RepoStatus{
					Context: stringPtr(fmt.Sprintf("job-%d", i)),
					State:   stringPtr(successState),
				}

				checkRuns[i] = &github.CheckRun{
					Name:       stringPtr(fmt.Sprintf("job-%d", i)),
					Status:     stringPtr(checkRunCompletedStatus),
					Conclusion: stringPtr(checkRunNeutralConclusion),
				}

				expectedGhaStatuses[i] = &ghaStatus{
					Job:   fmt.Sprintf("job-%d", i),
					State: successState,
				}
			}

			c := &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					start := (opts.Page - 1) * opts.PerPage
					if start >= len(statuses) {
						return &github.CombinedStatus{TotalCount: &num_statuses}, &github.Response{}, nil
					}
					max := min(opts.Page*opts.PerPage, len(statuses))
					sts := statuses[start:max]
					totalCount := len(statuses)
					resp := &github.Response{}
					if max < len(statuses) {
						resp.NextPage = opts.Page + 1
					}
					return &github.CombinedStatus{
						Statuses:   sts,
						TotalCount: &totalCount,
					}, resp, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					l := len(checkRuns)
					return &github.ListCheckRunsResults{
						CheckRuns: checkRuns,
						Total:     &l,
					}, nil, nil
				},
			}
			return test{
				fields: fields{
					client:      c,
					selfJobName: "self-job",
					owner:       "test-owner",
					repo:        "test-repo",
					ref:         "main",
				},
				wantErr: false,
				want:    expectedGhaStatuses,
			}
		}(),
		"succeeds to retrieve 162 statuses": func() test {
			num_statuses := 162
			statuses := make([]*github.RepoStatus, num_statuses)
			checkRuns := make([]*github.CheckRun, num_statuses)
			expectedGhaStatuses := make([]*ghaStatus, num_statuses)
			for i := 0; i < num_statuses; i++ {
				statuses[i] = &github.RepoStatus{
					Context: stringPtr(fmt.Sprintf("job-%d", i)),
					State:   stringPtr(successState),
				}

				checkRuns[i] = &github.CheckRun{
					Name:       stringPtr(fmt.Sprintf("job-%d", i)),
					Status:     stringPtr(checkRunCompletedStatus),
					Conclusion: stringPtr(checkRunNeutralConclusion),
				}

				expectedGhaStatuses[i] = &ghaStatus{
					Job:   fmt.Sprintf("job-%d", i),
					State: successState,
				}
			}

			c := &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					start := (opts.Page - 1) * opts.PerPage
					if start >= len(statuses) {
						return &github.CombinedStatus{TotalCount: &num_statuses}, &github.Response{}, nil
					}
					max := min(opts.Page*opts.PerPage, len(statuses))
					sts := statuses[start:max]
					totalCount := len(statuses)
					resp := &github.Response{}
					if max < len(statuses) {
						resp.NextPage = opts.Page + 1
					}
					return &github.CombinedStatus{
						Statuses:   sts,
						TotalCount: &totalCount,
					}, resp, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					l := len(checkRuns)
					return &github.ListCheckRunsResults{
						CheckRuns: checkRuns,
						Total:     &l,
					}, nil, nil
				},
			}
			return test{
				fields: fields{
					client:      c,
					selfJobName: "self-job",
					owner:       "test-owner",
					repo:        "test-repo",
					ref:         "main",
				},
				wantErr: false,
				want:    expectedGhaStatuses,
			}
		}(),
		"succeeds to retrieve 440 statuses": func() test {
			num_statuses := 440
			statuses := make([]*github.RepoStatus, num_statuses)
			checkRuns := make([]*github.CheckRun, num_statuses)
			expectedGhaStatuses := make([]*ghaStatus, num_statuses)
			for i := 0; i < num_statuses; i++ {
				statuses[i] = &github.RepoStatus{
					Context: stringPtr(fmt.Sprintf("job-%d", i)),
					State:   stringPtr(successState),
				}

				checkRuns[i] = &github.CheckRun{
					Name:       stringPtr(fmt.Sprintf("job-%d", i)),
					Status:     stringPtr(checkRunCompletedStatus),
					Conclusion: stringPtr(checkRunNeutralConclusion),
				}

				expectedGhaStatuses[i] = &ghaStatus{
					Job:   fmt.Sprintf("job-%d", i),
					State: successState,
				}
			}

			c := &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					start := (opts.Page - 1) * opts.PerPage
					if start >= len(statuses) {
						return &github.CombinedStatus{TotalCount: &num_statuses}, &github.Response{}, nil
					}
					max := min(opts.Page*opts.PerPage, len(statuses))
					sts := statuses[start:max]
					totalCount := len(statuses)
					resp := &github.Response{}
					if max < len(statuses) {
						resp.NextPage = opts.Page + 1
					}
					return &github.CombinedStatus{
						Statuses:   sts,
						TotalCount: &totalCount,
					}, resp, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					l := len(checkRuns)
					return &github.ListCheckRunsResults{
						CheckRuns: checkRuns,
						Total:     &l,
					}, nil, nil
				},
			}
			return test{
				fields: fields{
					client:      c,
					selfJobName: "self-job",
					owner:       "test-owner",
					repo:        "test-repo",
					ref:         "main",
				},
				wantErr: false,
				want:    expectedGhaStatuses,
			}
		}(),
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			sv := &statusValidator{
				repo:                         tt.fields.repo,
				owner:                        tt.fields.owner,
				ref:                          tt.fields.ref,
				selfJobName:                  tt.fields.selfJobName,
				ignoreDynamicGitHubWorkflows: tt.ignoreDynamicGitHubWorkflows,
				client:                       tt.fields.client,
			}
			got, err := sv.listGhaStatuses(tt.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("statusValidator.listStatuses() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got, want := len(got), len(tt.want); got != want {
				t.Errorf("statusValidator.listStatuses() length = %v, want %v", got, want)
			}
			for i := range tt.want {
				if !reflect.DeepEqual(got[i], tt.want[i]) {
					t.Errorf("statusValidator.listStatuses() - %d = %v, want %v", i, got[i], tt.want[i])
				}
			}
			if tt.workflowLookupCalls != nil && *tt.workflowLookupCalls != tt.wantWorkflowLookupCalls {
				t.Errorf("statusValidator.listStatuses() workflow lookup calls = %d, want %d", *tt.workflowLookupCalls, tt.wantWorkflowLookupCalls)
			}
		})
	}
}
