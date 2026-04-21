package status

import "fmt"

type status struct {
	totalJobs          []string
	completeJobs       []string
	errJobs            []string
	ignoredJobs        []string
	requiredJobs       []string
	unseenRequiredJobs []string
	succeeded          bool
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
	totalJobCount := len(s.totalJobs) + len(s.unseenRequiredJobs)
	incompleteJobCount := len(s.getIncompleteJobs()) + len(s.unseenRequiredJobs)
	completedJobCount := len(s.completeJobs)

	result := fmt.Sprintf(
		`%d out of %d

Total job count:       %d
Completed job count:   %d
Incompleted job count: %d
Failed job count:      %d
Ignored job count:     %d
Required job count:    %d
Unseen required count: %d
`,
		completedJobCount, totalJobCount,
		totalJobCount,
		completedJobCount,
		incompleteJobCount,
		len(s.errJobs),
		len(s.ignoredJobs),
		len(s.requiredJobs),
		len(s.unseenRequiredJobs),
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

::group::Unseen required jobs
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
		prettyPrintJobList(s.unseenRequiredJobs),
		prettyPrintJobList(s.totalJobs),
	)

	return result
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
