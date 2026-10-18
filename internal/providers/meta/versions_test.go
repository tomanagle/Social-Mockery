package meta

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVersionDispatchIsIndependent(t *testing.T) {
	p := &Provider{routers: map[string]http.Handler{}}
	for _, version := range []string{"v24.0", "v25.0"} {
		p.routers[version] = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Header().Set("Selected-Version", version) })
	}
	for _, version := range []string{"v24.0", "v25.0"} {
		for _, host := range []string{"graph.instagram.com", "graph.facebook.com"} {
			r := httptest.NewRequest("GET", "/https://"+host+"/"+version+"/123", nil)
			w := httptest.NewRecorder()
			p.ServeHTTP(w, r)
			if w.Code != 200 || w.Header().Get("Selected-Version") != version {
				t.Fatalf("dispatch %s: %d %s", version, w.Code, w.Header())
			}
		}
	}
	for _, version := range []string{"v23.0", "v26.0", "latest", "v25.1"} {
		w := httptest.NewRecorder()
		p.ServeHTTP(w, httptest.NewRequest("GET", "/https://graph.instagram.com/"+version+"/123", nil))
		if w.Code != 404 {
			t.Fatalf("unsupported version %s: %d", version, w.Code)
		}
	}
}

// A version's generated router must reject the other version even if the outer
// dispatch is accidentally wired to it. The paths themselves are versioned.
func TestGeneratedRoutersRejectOtherVersions(t *testing.T) {
	routers := versionRouters(&Provider{})
	for version, router := range routers {
		other := "v24.0"
		if version == other {
			other = "v25.0"
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("GET", "/"+other+"/123", nil))
		if w.Code != 404 {
			t.Fatalf("%s router accepted %s: %d", version, other, w.Code)
		}
	}
}
