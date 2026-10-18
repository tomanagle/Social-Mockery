package simulation

import "slices"

// PageTaskAllows checks the task subset implemented by the simulator.
func PageTaskAllows(tasks []string, scope string) bool {
	if slices.Contains(tasks, "MANAGE") || slices.Contains(tasks, "PROFILE_PLUS_FULL_CONTROL") {
		return true
	}
	var allowed []string
	switch scope {
	case "instagram_basic":
		allowed = []string{"ADVERTISE", "CREATE_CONTENT", "ANALYZE", "MESSAGING", "MODERATE", "PROFILE_PLUS_ADVERTISE", "PROFILE_PLUS_CREATE_CONTENT", "PROFILE_PLUS_ANALYZE", "PROFILE_PLUS_MESSAGING", "PROFILE_PLUS_MODERATE"}
	case "instagram_content_publish", "pages_manage_posts":
		allowed = []string{"CREATE_CONTENT", "PROFILE_PLUS_CREATE_CONTENT"}
	case "instagram_manage_comments", "pages_manage_engagement":
		allowed = []string{"MODERATE", "PROFILE_PLUS_MODERATE"}
	case "instagram_manage_messages":
		allowed = []string{"MESSAGING", "PROFILE_PLUS_MESSAGING"}
	case "instagram_manage_insights", "read_insights":
		allowed = []string{"ANALYZE", "PROFILE_PLUS_ANALYZE"}
	default:
		return true
	}
	for _, t := range allowed {
		if slices.Contains(tasks, t) {
			return true
		}
	}
	return false
}
