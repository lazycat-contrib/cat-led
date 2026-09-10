package web

import (
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"strings"
	"testing"
)

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
