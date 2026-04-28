package status

import (
	"fmt"
	"strings"
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
		ss := strings.Split(names, ",")
		for index, name := range ss {
			jobName := strings.TrimSpace(name)
			if len(jobName) == 0 {
				s.optionErrs = append(s.optionErrs, fmt.Errorf("ignored jobs contains empty entry at position %d", index+1))
				continue
			}
			jobs = append(jobs, jobName)
		}
		if len(s.optionErrs) != 0 {
			return
		}
		s.ignoredJobs = jobs
	}
}
