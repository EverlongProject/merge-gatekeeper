package status

import (
	"reflect"
	"testing"
)

func TestWithIgnoredJobsDropsDuplicates(t *testing.T) {
	sv := &statusValidator{}
	WithIgnoredJobs("job-01, job-02,job-01 ,job-02")(sv)

	want := []string{"job-01", "job-02"}
	if !reflect.DeepEqual(sv.ignoredJobs, want) {
		t.Errorf("ignoredJobs = %v, want %v", sv.ignoredJobs, want)
	}
}

func TestWithIgnoredJobsDropsDuplicatesIgnoringCase(t *testing.T) {
	sv := &statusValidator{}
	WithIgnoredJobs("Job-01,job-01,JOB-01")(sv)

	want := []string{"Job-01"}
	if !reflect.DeepEqual(sv.ignoredJobs, want) {
		t.Errorf("ignoredJobs = %v, want %v", sv.ignoredJobs, want)
	}
}
