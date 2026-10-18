package simulation

import "testing"

func TestPageTaskAllows(t *testing.T) {
	for _, tc := range []struct {
		task, scope string
		want        bool
	}{{"", "instagram_basic", false}, {"ADVERTISE", "instagram_basic", true}, {"ADVERTISE", "instagram_content_publish", false}, {"CREATE_CONTENT", "instagram_content_publish", true}, {"PROFILE_PLUS_CREATE_CONTENT", "instagram_content_publish", true}, {"MODERATE", "instagram_manage_comments", true}, {"ANALYZE", "instagram_manage_comments", false}, {"PROFILE_PLUS_FULL_CONTROL", "instagram_content_publish", true}, {"MANAGE", "instagram_manage_insights", true}} {
		t.Run(tc.task+"/"+tc.scope, func(t *testing.T) {
			if got := PageTaskAllows([]string{tc.task}, tc.scope); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
