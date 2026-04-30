package status

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"league.dev/merge-gatekeeper/internal/github"
	"league.dev/merge-gatekeeper/internal/validators"
)

const (
	successState = "success"
	errorState   = "error"
	failureState = "failure"
	pendingState = "pending"
)

// NOTE: https://docs.github.com/en/rest/reference/checks
const (
	checkRunCompletedStatus = "completed"
)
const (
	checkRunNeutralConclusion = "neutral"
	checkRunSuccessConclusion = "success"
	checkRunSkipConclusion    = "skipped"
)

const (
	maxStatusesPerPage  = 100
	maxCheckRunsPerPage = 100
)

const githubAdvancedSecurityAppSlug = "github-advanced-security"

var (
	ErrInvalidCombinedStatusResponse = errors.New("github combined status response is invalid")
	ErrInvalidCheckRunResponse       = errors.New("github checkRun response is invalid")
)

type ghaStatus struct {
	Job                        string
	State                      string
	IgnoredDynamicWorkflowName string
	IgnoredDynamicWorkflowPath string
	FailedLookupCommand        string
}

type statusValidator struct {
	repo                         string
	owner                        string
	ref                          string
	selfJobName                  string
	ignoredJobs                  []string
	ignoreDynamicGitHubWorkflows bool
	optionErrs                   []error
	ignoredJobsErrs              []error
	workflowRunByCheckSuiteCache map[int64]*github.WorkflowRun
	ignoredDynamicWorkflowByJob  map[string]ignoredDynamicWorkflowGroup
	client                       github.Client
}

func CreateValidator(c github.Client, opts ...Option) (validators.Validator, error) {
	sv := &statusValidator{
		client:                       c,
		ignoreDynamicGitHubWorkflows: true,
	}
	for _, opt := range opts {
		opt(sv)
	}
	if err := sv.validateFields(); err != nil {
		return nil, err
	}
	return sv, nil
}

func (sv *statusValidator) Name() string {
	return sv.selfJobName
}

func (sv *statusValidator) validateFields() error {
	errs := make([]error, 0, 7)

	if len(sv.repo) == 0 {
		errs = append(errs, errors.New("repository name is empty"))
	}
	if len(sv.owner) == 0 {
		errs = append(errs, errors.New("repository owner is empty"))
	}
	if len(sv.ref) == 0 {
		errs = append(errs, errors.New("reference of repository is empty"))
	}
	if len(sv.selfJobName) == 0 {
		errs = append(errs, errors.New("self job name is empty"))
	}
	if sv.client == nil {
		errs = append(errs, errors.New("github client is empty"))
	}
	errs = append(errs, sv.optionErrs...)
	errs = append(errs, sv.ignoredJobsErrs...)

	if len(errs) != 0 {
		return errors.Join(errs...)
	}

	return nil
}

func (sv *statusValidator) Validate(ctx context.Context) (validators.Status, error) {
	ghaStatuses, err := sv.listGhaStatuses(ctx)
	if err != nil {
		return nil, err
	}

	st := &status{
		totalJobs:    make([]string, 0, len(ghaStatuses)),
		completeJobs: make([]string, 0, len(ghaStatuses)),
		errJobs:      make([]string, 0, len(ghaStatuses)/2),
		ignoredJobs:  make([]string, 0, len(ghaStatuses)),
		succeeded:    true,
	}

	st.ignoredJobs = append(st.ignoredJobs, sv.ignoredJobs...)

	var successCnt int
	for _, ghaStatus := range ghaStatuses {
		if ghaStatus.IgnoredDynamicWorkflowPath != "" {
			successCnt++
			st.addIgnoredDynamicWorkflow(ghaStatus.IgnoredDynamicWorkflowName, ghaStatus.IgnoredDynamicWorkflowPath, ghaStatus.Job)
			continue
		}

		var toIgnore bool
		for _, ignored := range sv.ignoredJobs {
			if ghaStatus.Job == ignored {
				toIgnore = true
				break
			}
		}

		// Ignored jobs and this job itself should be considered as success regardless of their statuses.
		if toIgnore || ghaStatus.Job == sv.selfJobName {
			successCnt++
			continue
		}

		st.totalJobs = append(st.totalJobs, ghaStatus.Job)
		if ghaStatus.FailedLookupCommand != "" {
			st.failedLookupChecks = append(st.failedLookupChecks, failedLookupCheck{
				Job:     ghaStatus.Job,
				Command: ghaStatus.FailedLookupCommand,
			})
		}

		switch ghaStatus.State {
		case successState:
			st.completeJobs = append(st.completeJobs, ghaStatus.Job)
			successCnt++
		case errorState, failureState:
			st.errJobs = append(st.errJobs, ghaStatus.Job)
		}
	}
	if len(st.errJobs) != 0 {
		return nil, errors.New(st.Detail())
	}

	if len(ghaStatuses) != successCnt {
		st.succeeded = false
		return st, nil
	}

	return st, nil
}

func (s *status) addIgnoredDynamicWorkflow(workflowName, workflowPath, job string) {
	for index, workflow := range s.ignoredDynamicWorkflows {
		if workflow.WorkflowName == workflowName && workflow.WorkflowPath == workflowPath {
			s.ignoredDynamicWorkflows[index].Jobs = append(s.ignoredDynamicWorkflows[index].Jobs, job)
			return
		}
	}

	s.ignoredDynamicWorkflows = append(s.ignoredDynamicWorkflows, ignoredDynamicWorkflowGroup{
		WorkflowName: workflowName,
		WorkflowPath: workflowPath,
		Jobs:         []string{job},
	})
}

func (sv *statusValidator) getCombinedStatus(ctx context.Context) ([]*github.RepoStatus, error) {
	var combined []*github.RepoStatus
	page := 1
	for {
		c, resp, err := sv.client.GetCombinedStatus(ctx, sv.owner, sv.repo, sv.ref, &github.ListOptions{PerPage: maxStatusesPerPage, Page: page})
		if err != nil {
			return nil, err
		}
		combined = append(combined, c.Statuses...)
		if resp == nil || resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}
	return combined, nil
}

func (sv *statusValidator) listCheckRunsForRef(ctx context.Context) ([]*github.CheckRun, error) {
	var runResults []*github.CheckRun
	page := 1
	for {
		cr, _, err := sv.client.ListCheckRunsForRef(ctx, sv.owner, sv.repo, sv.ref, &github.ListCheckRunsOptions{ListOptions: github.ListOptions{
			Page:    page,
			PerPage: maxCheckRunsPerPage,
		}})
		if err != nil {
			return nil, err
		}
		runResults = append(runResults, cr.CheckRuns...)
		if cr.GetTotal() <= len(runResults) {
			break
		}
		page++
	}
	return runResults, nil
}

func (sv *statusValidator) getWorkflowRunForCheckSuite(ctx context.Context, checkSuiteID int64, misses map[int64]struct{}) (*github.WorkflowRun, error) {
	if sv.workflowRunByCheckSuiteCache == nil {
		sv.workflowRunByCheckSuiteCache = make(map[int64]*github.WorkflowRun)
	}
	if cached, ok := sv.workflowRunByCheckSuiteCache[checkSuiteID]; ok {
		return cached, nil
	}
	if _, ok := misses[checkSuiteID]; ok {
		return nil, nil
	}

	runs, _, err := sv.client.ListRepositoryWorkflowRuns(ctx, sv.owner, sv.repo, &github.ListWorkflowRunsOptions{
		CheckSuiteID: checkSuiteID,
		ListOptions: github.ListOptions{
			PerPage: 1,
			Page:    1,
		},
	})
	if err != nil {
		return nil, err
	}
	if runs == nil || len(runs.WorkflowRuns) == 0 {
		misses[checkSuiteID] = struct{}{}
		return nil, nil
	}

	run := runs.WorkflowRuns[0]
	sv.workflowRunByCheckSuiteCache[checkSuiteID] = run
	return run, nil
}

func (sv *statusValidator) shouldLookupWorkflowPath(run *github.CheckRun) bool {
	if !sv.ignoreDynamicGitHubWorkflows || run.DetailsURL == nil {
		return false
	}
	if run.App != nil && run.App.Slug != nil && *run.App.Slug == githubAdvancedSecurityAppSlug {
		return false
	}

	detailsURL := *run.DetailsURL
	return strings.Contains(detailsURL, "github.com/") && (strings.Contains(detailsURL, "/actions/") || strings.Contains(detailsURL, "/runs/"))
}

func (sv *statusValidator) getWorkflowLookupCommand(checkSuiteID int64) string {
	return fmt.Sprintf("gh api --method GET repos/%s/%s/actions/runs -F check_suite_id=%d", sv.owner, sv.repo, checkSuiteID)
}

func (sv *statusValidator) listGhaStatuses(ctx context.Context) ([]*ghaStatus, error) {
	combined, err := sv.getCombinedStatus(ctx)
	if err != nil {
		return nil, err
	}

	// Because multiple jobs with the same name may exist when jobs are created dynamically by third-party tools, etc.,
	// only the latest job should be managed.
	currentJobs := make(map[string]struct{})

	ghaStatuses := make([]*ghaStatus, 0, len(combined))
	for _, s := range combined {
		if s.Context == nil || s.State == nil {
			return nil, fmt.Errorf("%w context: %v, status: %v", ErrInvalidCombinedStatusResponse, s.Context, s.State)
		}
		if _, ok := currentJobs[*s.Context]; ok {
			continue
		}
		currentJobs[*s.Context] = struct{}{}

		ghaStatuses = append(ghaStatuses, &ghaStatus{
			Job:   *s.Context,
			State: *s.State,
		})
	}

	runResults, err := sv.listCheckRunsForRef(ctx)
	if err != nil {
		return nil, err
	}

	workflowRunByCheckSuiteMisses := make(map[int64]struct{})
	if sv.ignoredDynamicWorkflowByJob == nil {
		sv.ignoredDynamicWorkflowByJob = make(map[string]ignoredDynamicWorkflowGroup)
	}

	for _, run := range runResults {
		if run.Name == nil || run.Status == nil {
			return nil, fmt.Errorf("%w name: %v, status: %v", ErrInvalidCheckRunResponse, run.Name, run.Status)
		}
		if _, ok := currentJobs[*run.Name]; ok {
			continue
		}
		currentJobs[*run.Name] = struct{}{}

		ghaStatus := &ghaStatus{
			Job: *run.Name,
		}
		if cachedGroup, ok := sv.ignoredDynamicWorkflowByJob[*run.Name]; ok {
			ghaStatus.IgnoredDynamicWorkflowName = cachedGroup.WorkflowName
			ghaStatus.IgnoredDynamicWorkflowPath = cachedGroup.WorkflowPath
		} else if sv.shouldLookupWorkflowPath(run) {
			if run.CheckSuite != nil && run.CheckSuite.ID != nil {
				checkSuiteID := *run.CheckSuite.ID
				workflowRun, lookupErr := sv.getWorkflowRunForCheckSuite(ctx, checkSuiteID, workflowRunByCheckSuiteMisses)
				if lookupErr != nil || workflowRun == nil || workflowRun.Path == nil || len(*workflowRun.Path) == 0 {
					ghaStatus.FailedLookupCommand = sv.getWorkflowLookupCommand(checkSuiteID)
				} else if strings.HasPrefix(*workflowRun.Path, "dynamic/") {
					workflowName := *workflowRun.Path
					if workflowRun.Name != nil && len(*workflowRun.Name) != 0 {
						workflowName = *workflowRun.Name
					}
					ghaStatus.IgnoredDynamicWorkflowName = workflowName
					ghaStatus.IgnoredDynamicWorkflowPath = *workflowRun.Path
					sv.ignoredDynamicWorkflowByJob[*run.Name] = ignoredDynamicWorkflowGroup{
						WorkflowName: workflowName,
						WorkflowPath: *workflowRun.Path,
					}
				}
			}
		}

		if *run.Status != checkRunCompletedStatus {
			ghaStatus.State = pendingState
			ghaStatuses = append(ghaStatuses, ghaStatus)
			continue
		}
		if run.Conclusion == nil {
			return nil, fmt.Errorf("%w name: %v, status: %v, conclusion: %v", ErrInvalidCheckRunResponse, run.Name, run.Status, run.Conclusion)
		}

		switch *run.Conclusion {
		case checkRunNeutralConclusion, checkRunSuccessConclusion:
			ghaStatus.State = successState
		case checkRunSkipConclusion:
			continue
		default:
			ghaStatus.State = errorState
		}
		ghaStatuses = append(ghaStatuses, ghaStatus)
	}

	return ghaStatuses, nil
}
