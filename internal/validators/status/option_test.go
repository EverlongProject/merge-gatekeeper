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
