package sso

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"helios-auth-service/internal/models"
	"helios-auth-service/internal/session"
)

func TestProjectConfigCombinesLegacyCallbacks(t *testing.T) {
	p := models.Project{ProjectIDString: "nexus", ProjectName: "Nexus"}
	clients := []Client{
		{ID: "nexus-local", Name: "Legacy local", Enabled: true, RedirectURIs: pq.StringArray{"http://localhost:5173/callback"}},
		{ID: "nexus-prod", Name: "Legacy production", Enabled: true, RedirectURIs: pq.StringArray{"https://nexus.example/callback", "http://localhost:5173/callback"}},
	}
	got := summarizeConfig(p, clients)
	if got.ClientID != p.ProjectIDString || got.Name != p.ProjectName || !got.Configured || !got.Enabled || got.MixedEnabled || got.CredentialReady {
		t.Fatalf("unexpected project config: %+v", got)
	}
	if !reflect.DeepEqual(got.RedirectURIs, []string{"http://localhost:5173/callback", "https://nexus.example/callback"}) {
		t.Fatalf("callbacks not combined: %v", got.RedirectURIs)
	}
	clients[1].Enabled = false
	got = summarizeConfig(p, clients)
	if !got.MixedEnabled || got.Enabled {
		t.Fatal("mixed legacy settings must require explicit enablement")
	}
	if clients[1].Enabled {
		t.Fatal("reading settings changed legacy state")
	}
}

func TestProjectConfigEmptyAndCanonical(t *testing.T) {
	p := models.Project{ProjectIDString: "nexus", ProjectName: "Nexus"}
	got := summarizeConfig(p, nil)
	if got.Configured || got.CredentialReady || got.RedirectURIs == nil {
		t.Fatalf("invalid empty state: %+v", got)
	}
	got = summarizeConfig(p, []Client{{ID: "nexus", Enabled: true}})
	if !got.CredentialReady {
		t.Fatal("canonical project credential not recognized")
	}
}

func TestProjectConfigRejectsCSRFAndInvalidCallbacksBeforeDB(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := Handler{Frontend: "https://helios.example"}
	r := gin.New()
	r.POST("/config", h.SaveProjectConfig)
	for _, tc := range []struct {
		origin, body string
		want         int
	}{
		{"https://evil.example", `{"enabled":true,"redirect_uris":["https://nexus.example/callback"]}`, 403},
		{"https://helios.example", `{"redirect_uris":[]}`, 400},
		{"https://helios.example", `{"redirect_uris":["https://user:pass@evil.example/callback"]}`, 400},
		{"https://helios.example", `{"redirect_uris":["http://public.example/callback"]}`, 400},
	} {
		req := httptest.NewRequest("POST", "/config", bytes.NewBufferString(tc.body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", tc.origin)
		req.Header.Set("X-CSRF-Token", "test-token")
		req.AddCookie(&http.Cookie{Name: session.CSRFCookie, Value: "test-token"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != tc.want {
			t.Fatalf("got %d; want %d", w.Code, tc.want)
		}
	}
}
