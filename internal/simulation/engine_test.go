package simulation

import (
	"github.com/tomanagle/Social-Mockery/internal/store"
	"testing"
)

func TestStatusBoundaries(t *testing.T) {
	c := store.Container{CreatedAt: 100, ReadyAt: 105}
	for _, tc := range []struct {
		now  int64
		want string
	}{{100, "IN_PROGRESS"}, {104, "IN_PROGRESS"}, {105, "FINISHED"}, {86499, "FINISHED"}, {86500, "EXPIRED"}} {
		if got := Status(c, tc.now); got != tc.want {
			t.Errorf("at %d: got %s want %s", tc.now, got, tc.want)
		}
	}
	c.PublishedID = "post"
	if got := Status(c, 999999); got != "PUBLISHED" {
		t.Errorf("published container became %s", got)
	}
}
