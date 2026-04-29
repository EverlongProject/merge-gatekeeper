package status

import (
	"testing"
)

func Test_status_Detail(t *testing.T) {
	tests := map[string]struct {
		s    *status
		want string
	}{
		"return detail when totalJobs and completeJobs and errJobs is not empty": {
			s: &status{
				totalJobs: []string{
					"job-1",
					"job-2",
					"job-3",
				},
				completeJobs: []string{
					"job-2",
				},
				errJobs: []string{
					"job-3",
				},
			},
			want: `1 out of 3

Total job count:       3
Completed job count:   1
Incompleted job count: 1
Failed job count:      1
Ignored job count:     0
Ignored dynamic workflow check count: 0
Failed lookup count:   0

::group::Failed jobs
- job-3
::endgroup::

::group::Completed jobs
- job-2
::endgroup::

::group::Incomplete jobs
- job-1
::endgroup::

::group::Ignored jobs
[]
::endgroup::

::group::Ignored dynamic workflows
[]
::endgroup::

::group::Check runs with failed lookups
[]
::endgroup::

::group::All jobs
- job-1
- job-2
- job-3
::endgroup::
`,
		},
		"return detail with ignored jobs input": {
			s: &status{
				totalJobs: []string{
					"job-1",
					"job-2",
					"job-3",
					"job-4",
				},
				completeJobs: []string{
					"job-2",
					"job-4",
				},
				errJobs: []string{
					"job-3",
				},
				ignoredJobs: []string{
					"job-4",
				},
			},
			want: `2 out of 4

Total job count:       4
Completed job count:   2
Incompleted job count: 1
Failed job count:      1
Ignored job count:     1
Ignored dynamic workflow check count: 0
Failed lookup count:   0

::group::Failed jobs
- job-3
::endgroup::

::group::Completed jobs
- job-2
- job-4
::endgroup::

::group::Incomplete jobs
- job-1
::endgroup::

::group::Ignored jobs
- job-4
::endgroup::

::group::Ignored dynamic workflows
[]
::endgroup::

::group::Check runs with failed lookups
[]
::endgroup::

::group::All jobs
- job-1
- job-2
- job-3
- job-4
::endgroup::
`,
		},
		"return detail when totalJobs and completeJobs is empty": {
			s: &status{
				totalJobs:    []string{},
				completeJobs: []string{},
			},
			want: `0 out of 0

Total job count:       0
Completed job count:   0
Incompleted job count: 0
Failed job count:      0
Ignored job count:     0
Ignored dynamic workflow check count: 0
Failed lookup count:   0

::group::Failed jobs
[]
::endgroup::

::group::Completed jobs
[]
::endgroup::

::group::Incomplete jobs
[]
::endgroup::

::group::Ignored jobs
[]
::endgroup::

::group::Ignored dynamic workflows
[]
::endgroup::

::group::Check runs with failed lookups
[]
::endgroup::

::group::All jobs
[]
::endgroup::
`,
		},
		"return detail with dynamic workflow groups and failed lookups": {
			s: &status{
				totalJobs:    []string{"job-1", "job-2"},
				completeJobs: []string{"job-2"},
				ignoredDynamicWorkflows: []ignoredDynamicWorkflowGroup{
					{
						WorkflowName: "Copilot code review",
						WorkflowPath: "dynamic/copilot-pull-request-reviewer/copilot-pull-request-reviewer",
						Jobs:         []string{"Cleanup artifacts", "Agent"},
					},
				},
				failedLookupChecks: []failedLookupCheck{
					{
						Job:     "Analyze (go)",
						Command: "gh api repos/test-owner/test-repo/actions/runs -F check_suite_id=456",
					},
				},
			},
			want: `1 out of 2

Total job count:       2
Completed job count:   1
Incompleted job count: 1
Failed job count:      0
Ignored job count:     0
Ignored dynamic workflow check count: 2
Failed lookup count:   1

::group::Failed jobs
[]
::endgroup::

::group::Completed jobs
- job-2
::endgroup::

::group::Incomplete jobs
- job-1
::endgroup::

::group::Ignored jobs
[]
::endgroup::

::group::Ignored dynamic workflows
- Copilot code review (dynamic/copilot-pull-request-reviewer/copilot-pull-request-reviewer): Agent, Cleanup artifacts
::endgroup::

::group::Check runs with failed lookups
- Analyze (go) -> gh api repos/test-owner/test-repo/actions/runs -F check_suite_id=456
::endgroup::

::group::All jobs
- job-1
- job-2
::endgroup::
`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.s.Detail()
			if got != tt.want {
				t.Errorf("status.Detail() didn't match\n  got:\n%s\n\n  want:\n%s", got, tt.want)
			}
		})
	}
}

func Test_status_IsSuccess(t *testing.T) {
	tests := map[string]struct {
		s    *status
		want bool
	}{
		"returns true when status succeeded": {
			s:    &status{succeeded: true},
			want: true,
		},
		"returns false when status did not succeed": {
			s:    &status{succeeded: false},
			want: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := tt.s.IsSuccess(); got != tt.want {
				t.Errorf("status.IsSuccess() = %v, want %v", got, tt.want)
			}
		})
	}
}
