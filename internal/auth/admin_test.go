package auth

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSignedSessionRejectsTampering(t *testing.T) {
	value := encodeSession("user", "USER")
	if id, ok := decodeSession(value); !ok || id.ID != "user" {
		t.Fatal("valid session rejected")
	}
	payload, signature, _ := strings.Cut(value, ".")
	if _, ok := decodeSession(payload + "x." + signature); ok {
		t.Fatal("tampered identity accepted")
	}
	r := gin.New()
	r.Use(SessionMiddleware())
	r.GET("/", func(c *gin.Context) {
		if UserID(c) != "" {
			t.Error("plaintext cookie accepted")
		}
		c.Status(200)
	})
	request := httptest.NewRequest("GET", "/", nil)
	request.AddCookie(&http.Cookie{Name: "user_id", Value: "admin"})
	request.AddCookie(&http.Cookie{Name: "user_role", Value: "ADMIN"})
	r.ServeHTTP(httptest.NewRecorder(), request)
}
func TestAdminAndOriginGuards(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, test := range []struct {
		name, id, origin, marker string
		admin                    bool
		err                      error
		status                   int
	}{
		{name: "anonymous", status: 401}, {name: "normal", id: "user", status: 403}, {name: "lookup failure", id: "admin", err: errors.New("offline"), status: 503},
		{name: "cross origin", id: "admin", admin: true, origin: "https://evil.test", marker: "1", status: 403},
		{name: "missing marker", id: "admin", admin: true, status: 403},
		{name: "admin", id: "admin", admin: true, origin: "https://app.test", marker: "1", status: 204},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := gin.New()
			r.Use(SessionMiddleware(), RequireAdmin(func(context.Context, string) (bool, error) { return test.admin, test.err }), RequireSameOrigin())
			r.PUT("/", func(c *gin.Context) { c.Status(204) })
			request := httptest.NewRequest("PUT", "https://app.test/", nil)
			request.Header.Set("x-hc-user-id", test.id)
			request.Header.Set("x-hc-user-role", "ADMIN")
			request.Header.Set("Origin", test.origin)
			request.Header.Set("X-Cat-Led-Request", test.marker)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, request)
			if w.Code != test.status {
				t.Fatalf("%d %s", w.Code, w.Body.String())
			}
		})
	}
}
