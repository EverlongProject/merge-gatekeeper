package status

import (
	"fmt"
	"sort"
	"strings"
)

type status struct {
	totalJobs               []string
	completeJobs            []string
	errJobs                 []string
	ignoredJobs             []string
	ignoredDynamicWorkflows []ignoredDynamicWorkflowGroup
	failedLookupChecks      []failedLookupCheck
	succeeded               bool
}

type ignoredDynamicWorkflowGroup struct {
	WorkflowName string
	WorkflowPath string
	Jobs         []string
}

type failedLookupCheck struct {
	Job     string
	Command string
}

func prettyPrintJobList(jobs []string) string {
	result := ""
	if len(jobs) == 0 {
		result = "[]"
	}
	for i, job := range jobs {
		result += fmt.Sprintf("- %s", job)
		if i != len(jobs)-1 {
			result += "\n"
		}
	}

	return result
}

func (s *status) Detail() string {
	result := fmt.Sprintf(
		`%d out of %d

Total job count:       %d
Completed job count:   %d
Incompleted job count: %d
Failed job count:      %d
Ignored job count:     %d
Ignored dynamic workflow check count: %d
Failed lookup count:   %d
`,
		len(s.completeJobs), len(s.totalJobs),
		len(s.totalJobs),
		len(s.completeJobs),
		len(s.getIncompleteJobs()),
		len(s.errJobs),
		len(s.ignoredJobs),
		s.getIgnoredDynamicWorkflowCheckCount(),
		len(s.failedLookupChecks),
	)

	result = fmt.Sprintf(`%s
::group::Failed jobs
%s
::endgroup::

::group::Completed jobs
%s
::endgroup::

::group::Incomplete jobs
%s
::endgroup::

::group::Ignored jobs
%s
::endgroup::

::group::Ignored dynamic workflows
%s
::endgroup::

::group::Check runs with failed lookups
%s
::endgroup::

::group::All jobs
%s
::endgroup::
`,
		result,
		prettyPrintJobList(s.errJobs),
		prettyPrintJobList(s.completeJobs),
		prettyPrintJobList(s.getIncompleteJobs()),
		prettyPrintJobList(s.ignoredJobs),
		s.formatIgnoredDynamicWorkflows(),
		s.formatFailedLookupChecks(),
		prettyPrintJobList(s.totalJobs),
	)

	return result
}

func (s *status) getIgnoredDynamicWorkflowCheckCount() int {
	count := 0
	for _, workflow := range s.ignoredDynamicWorkflows {
		count += len(workflow.Jobs)
	}
	return count
}

func (s *status) formatIgnoredDynamicWorkflows() string {
	if len(s.ignoredDynamicWorkflows) == 0 {
		return "[]"
	}

	groups := append([]ignoredDynamicWorkflowGroup(nil), s.ignoredDynamicWorkflows...)
	sort.Slice(groups, func(i, j int) bool {
		left := groups[i].WorkflowName + groups[i].WorkflowPath
		right := groups[j].WorkflowName + groups[j].WorkflowPath
		return left < right
	})

	lines := make([]string, 0, len(groups))
	for _, group := range groups {
		jobs := append([]string(nil), group.Jobs...)
		sort.Strings(jobs)
		lines = append(lines, fmt.Sprintf("- %s (%s): %s", group.WorkflowName, group.WorkflowPath, strings.Join(jobs, ", ")))
	}

	return strings.Join(lines, "\n")
}

func (s *status) formatFailedLookupChecks() string {
	if len(s.failedLookupChecks) == 0 {
		return "[]"
	}

	checks := append([]failedLookupCheck(nil), s.failedLookupChecks...)
	sort.Slice(checks, func(i, j int) bool {
		return checks[i].Job < checks[j].Job
	})

	lines := make([]string, 0, len(checks))
	for _, check := range checks {
		if check.Command == "" {
			lines = append(lines, fmt.Sprintf("- %s", check.Job))
			continue
		}
		lines = append(lines, fmt.Sprintf("- %s -> %s", check.Job, check.Command))
	}

	return strings.Join(lines, "\n")
}

func (s *status) IsSuccess() bool {
	// TDOO: Add test case
	return s.succeeded
}

func (s *status) getIncompleteJobs() []string {
	var incomplete []string

	for _, job := range s.totalJobs {
		found := false
		for _, complete := range s.completeJobs {
			if job == complete {
				found = true
				break
			}
		}

		for _, failed := range s.errJobs {
			if job == failed {
				found = true
				break
			}
		}

		for _, ignored := range s.ignoredJobs {
			if job == ignored {
				found = true
				break
			}
		}
		if !found {
			incomplete = append(incomplete, job)
		}
	}
	return incomplete
}
