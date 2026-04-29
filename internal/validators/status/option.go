package status

import (
	"fmt"
	"strings"

	"league.dev/merge-gatekeeper/internal/multierror"
)

type Option func(s *statusValidator)

func WithSelfJob(name string) Option {
	return func(s *statusValidator) {
		if len(name) != 0 {
			s.selfJobName = name
		}
	}
}

func WithGitHubOwnerAndRepo(owner, repo string) Option {
	return func(s *statusValidator) {
		if len(owner) != 0 {
			s.owner = owner
		}
		if len(repo) != 0 {
			s.repo = repo
		}
	}
}

func WithGitHubRef(ref string) Option {
	return func(s *statusValidator) {
		if len(ref) != 0 {
			s.ref = ref
		}
	}
}

func WithIgnoredJobs(names string) Option {
	return func(s *statusValidator) {
		if len(names) == 0 {
			return
		}

		jobs := []string{}
		errs := make(multierror.Errors, 0)
		ss := strings.Split(names, ",")
		for index, name := range ss {
			jobName := strings.TrimSpace(name)
			if len(jobName) == 0 {
				errs = append(errs, fmt.Errorf("ignored jobs contains empty entry at position %d", index+1))
				continue
			}
			jobs = append(jobs, jobName)
		}
		s.ignoredJobsErrs = errs
		if len(errs) != 0 {
			return
		}
		s.ignoredJobs = jobs
	}
}

func WithIgnoreDynamicGitHubWorkflows(ignore bool) Option {
	return func(s *statusValidator) {
		s.ignoreDynamicGitHubWorkflows = ignore
	}
}
