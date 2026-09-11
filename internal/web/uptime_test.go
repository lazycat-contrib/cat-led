package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestReadSystemUptime(t *testing.T) {
	for _, tc := range []struct {
		input   string
		want    int64
		invalid bool
	}{
		{"3720.99 99999.00\n", 3720, false},
		{"0.00 0.00\n", 0, false},
		{"183480.50 1\n", 183480, false},
		{"", 0, true}, {"oops 1", 0, true}, {"-1 0", 0, true},
		{"NaN 0", 0, true}, {"Inf 0", 0, true}, {"1e30 0", 0, true},
	} {
		t.Run(tc.input, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "uptime")
			if err := os.WriteFile(path, []byte(tc.input), 0600); err != nil {
				t.Fatal(err)
			}
			got, err := readSystemUptime(path)
			if (err != nil) != tc.invalid || got != tc.want {
				t.Fatalf("got %d, %v; want %d, invalid=%t", got, err, tc.want, tc.invalid)
			}
		})
	}
	if _, err := readSystemUptime(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("missing uptime file should fail")
	}
}

func TestSystemUptimeRoute(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("requires Linux procfs")
	}
	gin.SetMode(gin.TestMode)
	server := &Server{engine: gin.New()}
	if err := server.SetupRoutes(); err != nil {
		t.Fatal(err)
	}
	for _, authenticated := range []bool{false, true} {
		request := httptest.NewRequest(http.MethodGet, "/api/system/uptime", nil)
		if authenticated {
			request.Header.Set("x-hc-user-id", "uptime-test")
		}
		w := httptest.NewRecorder()
		server.engine.ServeHTTP(w, request)
		if !authenticated {
			if w.Code == http.StatusOK {
				t.Fatal("anonymous request was accepted")
			}
			continue
		}
		if w.Code != http.StatusOK {
			t.Fatalf("status %d: %s", w.Code, w.Body.String())
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("uptime must not be cached")
		}
		var response struct {
			Seconds int64 `json:"uptime_seconds"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		actual, err := readSystemUptime("/proc/uptime")
		if err != nil {
			t.Fatal(err)
		}
		if response.Seconds <= 0 || actual-response.Seconds < 0 || actual-response.Seconds > 5 {
			t.Fatalf("response %d doesn't match system uptime %d", response.Seconds, actual)
		}
	}
}
