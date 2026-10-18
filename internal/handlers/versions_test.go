package handlers_test

import (
	"github.com/tomanagle/Social-Mockery/internal/profiles"
	"testing"
)

func TestPublishingEachVersion(t *testing.T) {
	for _, version := range []string{"v24.0", "v25.0"} {
		for _, host := range []string{"graph.instagram.com", "graph.facebook.com"} {
			t.Run(host+"/"+version, func(t *testing.T) {
				h := setup(t, profiles.Default(), 100, "")
				base := "/https://" + host + "/" + version + "/"
				id := object(t, request(h, "POST", base+account+"/media", "application/json", `{"image_url":"https://example.test/image.jpg"}`, "mock-token"), 200)["id"].(string)
				status := object(t, request(h, "GET", base+id+"?fields=status_code", "", "", "mock-token"), 200)
				if status["status_code"] != "IN_PROGRESS" {
					t.Fatalf("status: %v", status)
				}
				advance(t, h, 5)
				postID := object(t, request(h, "POST", base+account+"/media_publish", "application/x-www-form-urlencoded", "creation_id="+id, "mock-token"), 200)["id"].(string)
				post := object(t, request(h, "GET", base+postID+"?fields=media_type", "", "", "mock-token"), 200)
				if post["media_type"] != "IMAGE" {
					t.Fatalf("post: %v", post)
				}
			})
		}
	}
}
