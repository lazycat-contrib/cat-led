package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cat-led/internal/buildinfo"

	"github.com/gin-gonic/gin"
)

func TestHomeShowsBinaryVersion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := &Server{engine: gin.New()}
	if err := server.SetupRoutes(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("x-hc-user-id", "version-test")
	w := httptest.NewRecorder()
	server.engine.ServeHTTP(w, request)
	if w.Code != http.StatusOK {
		t.Fatalf("home status = %d", w.Code)
	}
	if want := `id="about-version">` + buildinfo.Version + `</dd>`; !strings.Contains(w.Body.String(), want) {
		t.Fatalf("home is missing binary version %q", buildinfo.Version)
	}
}

func TestLocalizedTemplatesRender(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := &Server{engine: gin.New()}
	if err := server.setupTemplates(); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name string
		data gin.H
	}{
		{"index.html", nil}, {"config.html", nil},
		{"login.html", gin.H{"OIDCAvailable": true, "OIDCLoginURL": "/auth/oidc/login"}},
		{"login.html", gin.H{"OIDCAvailable": false}},
	} {
		w := httptest.NewRecorder()
		if err := server.engine.HTMLRender.Instance(test.name, test.data).Render(w); err != nil {
			t.Fatal(err)
		}
		body := w.Body.String()
		if !strings.Contains(body, "/static/js/i18n.js") {
			t.Fatalf("missing localization script in %s", test.name)
		}
		wantToggle := test.name != "config.html"
		if strings.Contains(body, "data-language-toggle") != wantToggle {
			t.Fatalf("unexpected language control presence in %s", test.name)
		}
		if test.name == "config.html" && !strings.Contains(body, "{{.Name}} 任务执行成功") {
			t.Fatal("notification template placeholder was changed")
		}
	}
}
