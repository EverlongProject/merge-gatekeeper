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
		jobs, errs := parseJobList("ignored jobs", names)
		s.ignoredJobsErrs = errs
		if len(errs) != 0 {
			return
		}
		s.ignoredJobs = dedupeJobNames(jobs)
	}
}

// dedupeJobNames keeps the first spelling of each name. It ignores case,
// because ignored jobs match job names case-insensitively.
func dedupeJobNames(names []string) []string {
	jobs := make([]string, 0, len(names))
	seen := map[string]struct{}{}
	for _, name := range names {
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		jobs = append(jobs, name)
	}
	return jobs
}

// WithRequiredContexts names statuses or check runs that must exist on the ref
// before it can pass; an absent one keeps the validation pending.
func WithRequiredContexts(names string) Option {
	return func(s *statusValidator) {
		if len(names) == 0 {
			return
		}
		contexts, errs := parseJobList("required contexts", names)
		s.requiredContextsErrs = errs
		if len(errs) != 0 {
			return
		}
		s.requiredContexts = contexts
	}
}

func parseJobList(label, names string) ([]string, []error) {
	jobs := []string{}
	errs := make([]error, 0)
	for index, name := range strings.Split(names, ",") {
		jobName := strings.TrimSpace(name)
		if len(jobName) == 0 {
			errs = append(errs, fmt.Errorf("%s contains empty entry at position %d", label, index+1))
			continue
		}
		jobs = append(jobs, jobName)
	}
	return jobs, errs
}

func WithIgnoreDynamicGitHubWorkflows(ignore bool) Option {
	return func(s *statusValidator) {
		s.ignoreDynamicGitHubWorkflows = ignore
	}
}
