package cli

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	ghapi "github.com/google/go-github/v69/github"
	"github.com/spf13/cobra"

	"league.dev/merge-gatekeeper/internal/github"
	ghmock "league.dev/merge-gatekeeper/internal/github/mock"
	"league.dev/merge-gatekeeper/internal/validators"
	"league.dev/merge-gatekeeper/internal/validators/mock"
	"league.dev/merge-gatekeeper/internal/validators/status"
)

func TestMain(m *testing.M) {
	validateInvalSecond = 1
	timeoutSecond = 2
	successConfirmationPolls = 0
	os.Exit(m.Run())
}

func Test_ownerAndRepository(t *testing.T) {
	tests := map[string]struct {
		str       string
		wantOwner string
		wantRepo  string
	}{
		"returns empty when str is empty": {
			str:       "",
			wantOwner: "",
			wantRepo:  "",
		},
		"returns (owner, repo) when str is owner/repo": {
			str:       "owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		"returns (owner, '') when str is owner": {
			str:       "owner",
			wantOwner: "owner",
			wantRepo:  "",
		},
		"returns ('', repo) when str is /repo": {
			str:       "/repo",
			wantOwner: "",
			wantRepo:  "repo",
		},
		"returns (owner, repo/repo) when str is owner/repo/repo": {
			str:       "owner/repo/repo",
			wantOwner: "owner",
			wantRepo:  "repo/repo",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			gotOwner, gotRepo := ownerAndRepository(tt.str)
			if gotOwner != tt.wantOwner {
				t.Errorf("ownerAndRepository() owner = %s, wantOwner: %s", gotOwner, tt.wantOwner)
			}
			if gotRepo != tt.wantRepo {
				t.Errorf("ownerAndRepository() repo = %s, wantOwner: %s", gotRepo, tt.wantRepo)
			}
		})
	}
}

func Test_doValidateCmd(t *testing.T) {
	tests := map[string]struct {
		ctx                      context.Context
		cmd                      *cobra.Command
		vs                       []validators.Validator
		successConfirmationPolls uint
		timeoutSecond            uint
		wantErr                  bool
	}{
		"returns nil when the validation is success": {
			ctx: context.Background(),
			cmd: &cobra.Command{},
			vs: []validators.Validator{
				&mock.Validator{
					NameFunc: func() string { return "validator-1" },
					ValidateFunc: func(ctx context.Context) (validators.Status, error) {
						return &mock.Status{
							DetailFunc:    func() string { return "success-1" },
							IsSuccessFunc: func() bool { return true },
						}, nil
					},
				},
				&mock.Validator{
					NameFunc: func() string { return "validator-2" },
					ValidateFunc: func(ctx context.Context) (validators.Status, error) {
						return &mock.Status{
							DetailFunc:    func() string { return "success-2" },
							IsSuccessFunc: func() bool { return true },
						}, nil
					},
				},
			},
			successConfirmationPolls: 0,
			timeoutSecond:            2,
			wantErr:                  false,
		},
		"returns nil after required confirmation polls succeed": {
			ctx: context.Background(),
			cmd: &cobra.Command{},
			vs: []validators.Validator{
				&mock.Validator{
					NameFunc: func() string { return "validator-1" },
					ValidateFunc: func() func(ctx context.Context) (validators.Status, error) {
						calls := 0
						return func(ctx context.Context) (validators.Status, error) {
							calls++
							return &mock.Status{
								DetailFunc:    func() string { return "success" },
								IsSuccessFunc: func() bool { return true },
							}, nil
						}
					}(),
				},
			},
			successConfirmationPolls: 2,
			timeoutSecond:            5,
			wantErr:                  false,
		},
		"resets confirmation streak when a later poll is not successful": {
			ctx: context.Background(),
			cmd: &cobra.Command{},
			vs: []validators.Validator{
				&mock.Validator{
					NameFunc: func() string { return "validator-1" },
					ValidateFunc: func() func(ctx context.Context) (validators.Status, error) {
						calls := 0
						return func(ctx context.Context) (validators.Status, error) {
							calls++
							statuses := []bool{true, false, true, true}
							idx := calls - 1
							if idx >= len(statuses) {
								idx = len(statuses) - 1
							}
							ok := statuses[idx]
							return &mock.Status{
								DetailFunc:    func() string { return "status" },
								IsSuccessFunc: func() bool { return ok },
							}, nil
						}
					}(),
				},
			},
			successConfirmationPolls: 1,
			timeoutSecond:            5,
			wantErr:                  false,
		},
		"returns error when the validation timed out": {
			ctx: context.Background(),
			cmd: &cobra.Command{},
			vs: []validators.Validator{
				&mock.Validator{
					NameFunc: func() string { return "validator-1" },
					ValidateFunc: func(ctx context.Context) (validators.Status, error) {
						return &mock.Status{
							DetailFunc:    func() string { return "fails-1" },
							IsSuccessFunc: func() bool { return false },
						}, nil
					},
				},
				&mock.Validator{
					NameFunc: func() string { return "validator-2" },
					ValidateFunc: func(ctx context.Context) (validators.Status, error) {
						return &mock.Status{
							DetailFunc:    func() string { return "fails-2" },
							IsSuccessFunc: func() bool { return false },
						}, nil
					},
				},
			},
			successConfirmationPolls: 0,
			timeoutSecond:            2,
			wantErr:                  true,
		},
		"returns error when the validator return an error": {
			ctx: context.Background(),
			cmd: &cobra.Command{},
			vs: []validators.Validator{
				&mock.Validator{
					NameFunc: func() string { return "validator-1" },
					ValidateFunc: func(ctx context.Context) (validators.Status, error) {
						return nil, errors.New("err")
					},
				},
			},
			successConfirmationPolls: 0,
			timeoutSecond:            2,
			wantErr:                  true,
		},
		"returns error when success confirmation polls is too large": {
			ctx:                      context.Background(),
			cmd:                      &cobra.Command{},
			vs:                       []validators.Validator{},
			successConfirmationPolls: ^uint(0),
			timeoutSecond:            2,
			wantErr:                  true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			prevSuccessConfirmationPolls := successConfirmationPolls
			prevTimeoutSecond := timeoutSecond
			successConfirmationPolls = tt.successConfirmationPolls
			timeoutSecond = tt.timeoutSecond
			defer func() {
				successConfirmationPolls = prevSuccessConfirmationPolls
				timeoutSecond = prevTimeoutSecond
			}()

			if err := doValidateCmd(tt.ctx, tt.cmd, tt.vs...); (err != nil) != tt.wantErr {
				t.Errorf("doValidateCmd() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_validateCmd_rejectsInvalidRequiredContexts(t *testing.T) {
	tests := map[string]struct {
		args    []string
		wantErr string
	}{
		"a context both required and ignored": {
			args:    []string{"--ignored=lint," + setupJob, "--required-contexts=CI/CircleCI: Setup_Branch_Config"},
			wantErr: "listed in both required contexts and ignored jobs",
		},
		"a required context that is the default self job": {
			args:    []string{"--required-contexts=lint,Merge-Gatekeeper"},
			wantErr: "is the self job and cannot be a required context",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// validateCmd resets every flag-backed global to its default; restore them for later tests.
			prevRepo, prevRef, prevSelf := ghRepo, ghRef, selfJobName
			prevTimeout, prevInterval, prevPolls := timeoutSecond, validateInvalSecond, successConfirmationPolls
			prevIgnored, prevRequired, prevDynamic := ignoredJobs, requiredContexts, ignoreDynamicGitHubWorkflows
			defer func() {
				ghRepo, ghRef, selfJobName = prevRepo, prevRef, prevSelf
				timeoutSecond, validateInvalSecond, successConfirmationPolls = prevTimeout, prevInterval, prevPolls
				ignoredJobs, requiredContexts, ignoreDynamicGitHubWorkflows = prevIgnored, prevRequired, prevDynamic
			}()

			cmd := validateCmd()
			cmd.SetArgs(append([]string{"--ref=sha", "--repo=owner/repo"}, tt.args...))
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)

			start := time.Now()
			err := cmd.ExecuteContext(context.Background())
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("validate error = %v, want %q", err, tt.wantErr)
			}
			// The configuration error must stop the command before the first poll reaches GitHub.
			if !strings.Contains(err.Error(), "failed to create validator") {
				t.Errorf("validate error = %v, want it raised while creating the validator", err)
			}
			if elapsed := time.Since(start); elapsed > time.Second {
				t.Errorf("validate took %s, want an immediate configuration error", elapsed)
			}
		})
	}
}

// ghPoll is what the fake GitHub API returns for one poll of the ref.
type ghPoll struct {
	statuses  map[string]string // context -> state
	checkRuns map[string]string // name -> conclusion; "" means in progress
}

// pollingClient serves polls in order and repeats the last one.
type pollingClient struct {
	ghmock.Client
	polls []ghPoll
	calls int
}

func newPollingClient(polls ...ghPoll) *pollingClient {
	c := &pollingClient{polls: polls}
	c.GetCombinedStatusFunc = func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
		c.calls++
		p := c.current()
		out := &github.CombinedStatus{}
		for name, state := range p.statuses {
			out.Statuses = append(out.Statuses, &github.RepoStatus{Context: ghapi.String(name), State: ghapi.String(state)})
		}
		return out, nil, nil
	}
	c.ListCheckRunsForRefFunc = func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
		p := c.current()
		out := &github.ListCheckRunsResults{}
		for name, conclusion := range p.checkRuns {
			run := &github.CheckRun{Name: ghapi.String(name), Status: ghapi.String("in_progress")}
			if conclusion != "" {
				run.Status = ghapi.String("completed")
				run.Conclusion = ghapi.String(conclusion)
			}
			out.CheckRuns = append(out.CheckRuns, run)
		}
		out.Total = ghapi.Int(len(out.CheckRuns))
		return out, nil, nil
	}
	return c
}

func (c *pollingClient) current() ghPoll {
	idx := c.calls - 1
	if idx >= len(c.polls) {
		idx = len(c.polls) - 1
	}
	return c.polls[idx]
}

const setupJob = "ci/circleci: setup_branch_config"

func Test_doValidateCmd_requiredContexts(t *testing.T) {
	tests := map[string]struct {
		required                 string
		polls                    []ghPoll
		successConfirmationPolls uint
		timeoutSecond            uint
		wantErr                  error
		wantFailedJob            string // the job the poll-1 failure must name
		wantCalls                int    // 0 skips the check
		// wantErr and wantFailedJob both unset means success.
	}{
		"times out when a required status never appears although everything visible is green": {
			required: setupJob,
			polls: []ghPoll{
				{statuses: map[string]string{"lint": "success"}, checkRuns: map[string]string{"build": "success"}},
			},
			timeoutSecond: 2,
			wantErr:       context.DeadlineExceeded,
		},
		"times out when nothing is reported and a context is required": {
			required:      setupJob,
			polls:         []ghPoll{{}},
			timeoutSecond: 2,
			wantErr:       context.DeadlineExceeded,
		},
		"succeeds on the first poll for the same green ref when nothing is required": {
			polls: []ghPoll{
				{statuses: map[string]string{"lint": "success"}, checkRuns: map[string]string{"build": "success"}},
			},
			timeoutSecond: 2,
			wantCalls:     1,
		},
		"keeps polling while a required status is pending and succeeds once it is green": {
			required: setupJob,
			polls: []ghPoll{
				{statuses: map[string]string{"lint": "success", setupJob: "pending"}},
				{statuses: map[string]string{"lint": "success", setupJob: "pending"}},
				{statuses: map[string]string{"lint": "success", setupJob: "success"}},
			},
			timeoutSecond: 5,
			wantCalls:     3,
		},
		"a required check run counts as present and must be green": {
			required: "build",
			polls: []ghPoll{
				{statuses: map[string]string{"lint": "success"}},
				{statuses: map[string]string{"lint": "success"}, checkRuns: map[string]string{"build": ""}},
				{statuses: map[string]string{"lint": "success"}, checkRuns: map[string]string{"build": "success"}},
			},
			timeoutSecond: 5,
			wantCalls:     3,
		},
		"matches required names case-insensitively like ignored": {
			required: setupJob,
			polls: []ghPoll{
				{statuses: map[string]string{"CI/CircleCI: Setup_Branch_Config": "success"}},
			},
			timeoutSecond: 2,
			wantCalls:     1,
		},
		"fails without waiting for the timeout when a required status fails": {
			required: setupJob,
			polls: []ghPoll{
				{statuses: map[string]string{"lint": "success", setupJob: "failure"}},
			},
			timeoutSecond: 2,
			wantFailedJob: setupJob,
			wantCalls:     1,
		},
		"fails without waiting for the timeout when a required check run fails": {
			required: "build",
			polls: []ghPoll{
				{checkRuns: map[string]string{"build": "failure"}},
			},
			timeoutSecond: 2,
			wantFailedJob: "build",
			wantCalls:     1,
		},
		"times out when a required check run is skipped": {
			required: "build",
			polls: []ghPoll{
				{statuses: map[string]string{"lint": "success"}, checkRuns: map[string]string{"build": "skipped"}},
			},
			timeoutSecond: 2,
			wantErr:       context.DeadlineExceeded,
		},
		"a poll where the required status disappears resets the confirmation streak": {
			required: setupJob,
			polls: []ghPoll{
				{statuses: map[string]string{"lint": "success", setupJob: "success"}},
				{statuses: map[string]string{"lint": "success"}},
				{statuses: map[string]string{"lint": "success", setupJob: "success"}},
				{statuses: map[string]string{"lint": "success", setupJob: "success"}},
			},
			successConfirmationPolls: 1,
			timeoutSecond:            6,
			wantCalls:                4,
		},
		"a poll where the required status is not yet green resets the confirmation streak": {
			required: setupJob,
			polls: []ghPoll{
				{statuses: map[string]string{"lint": "success", setupJob: "success"}},
				{statuses: map[string]string{"lint": "success", setupJob: "pending"}},
				{statuses: map[string]string{"lint": "success", setupJob: "success"}},
				{statuses: map[string]string{"lint": "success", setupJob: "success"}},
			},
			successConfirmationPolls: 1,
			timeoutSecond:            6,
			wantCalls:                4,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			prevSuccessConfirmationPolls := successConfirmationPolls
			prevTimeoutSecond := timeoutSecond
			successConfirmationPolls = tt.successConfirmationPolls
			timeoutSecond = tt.timeoutSecond
			defer func() {
				successConfirmationPolls = prevSuccessConfirmationPolls
				timeoutSecond = prevTimeoutSecond
			}()

			client := newPollingClient(tt.polls...)
			v, err := status.CreateValidator(client,
				status.WithSelfJob(defaultSelfJobName),
				status.WithGitHubOwnerAndRepo("owner", "repo"),
				status.WithGitHubRef("sha"),
				status.WithRequiredContexts(tt.required),
			)
			if err != nil {
				t.Fatalf("CreateValidator() error = %v", err)
			}

			err = doValidateCmd(context.Background(), &cobra.Command{}, v)
			switch {
			case tt.wantErr != nil:
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("doValidateCmd() error = %v, want %v", err, tt.wantErr)
				}
			case tt.wantFailedJob != "":
				want := "::group::Failed jobs\n- " + tt.wantFailedJob + "\n::endgroup::"
				if err == nil || !strings.Contains(err.Error(), want) {
					t.Fatalf("doValidateCmd() error = %v, want the failed jobs group to name only %q", err, tt.wantFailedJob)
				}
			default:
				if err != nil {
					t.Fatalf("doValidateCmd() error = %v, want nil", err)
				}
			}
			if tt.wantCalls != 0 && client.calls != tt.wantCalls {
				t.Errorf("polls = %d, want %d", client.calls, tt.wantCalls)
			}
		})
	}
}
