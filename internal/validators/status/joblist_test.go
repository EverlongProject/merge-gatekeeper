package status

import "testing"

func TestPrettyPrintJobList(t *testing.T) {
	tests := map[string]struct {
		jobs []string
		want string
	}{
		"returns brackets for an empty list": {
			jobs: nil,
			want: "[]",
		},
		"returns one line for one job": {
			jobs: []string{"job-01"},
			want: "- job-01",
		},
		"returns one line per job without a trailing newline": {
			jobs: []string{"job-01", "job-02"},
			want: "- job-01\n- job-02",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := prettyPrintJobList(tt.jobs); got != tt.want {
				t.Errorf("prettyPrintJobList() = %q, want %q", got, tt.want)
			}
		})
	}
}
