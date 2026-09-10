package handlers

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"cat-led/internal/auth"
	"cat-led/internal/biz"
	"cat-led/internal/pkg/zlog"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func TestAccountPreferencesAndMigration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.db")
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	// Reproduce the table shipped before these preferences existed.
	_, err = db.Exec(`CREATE TABLE user_preferences (id integer PRIMARY KEY AUTOINCREMENT, user_id text NOT NULL, bulb_style text NOT NULL DEFAULT 'classic', created_at datetime NOT NULL, updated_at datetime NOT NULL);
 CREATE UNIQUE INDEX userpreference_user_id ON user_preferences(user_id);
 INSERT INTO user_preferences(user_id,bulb_style,created_at,updated_at) VALUES('alice','lava',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);`)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
	old := scheduleUseCase
	scheduleUseCase = biz.NewScheduleUseCase(path, &zlog.Logger{Logger: zerolog.New(io.Discard)})
	if scheduleUseCase == nil {
		t.Fatal("migration failed")
	}
	t.Cleanup(func() { scheduleUseCase.GetClient().Close(); scheduleUseCase = old })
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(auth.SessionMiddleware())
	router.GET("/preference", GetUserPreference)
	router.PUT("/preference", auth.RequireSameOrigin(), UpdateUserPreference)
	request := func(method, user, body string) (int, UserPreferenceResponse) {
		t.Helper()
		req := httptest.NewRequest(method, "http://app.test/preference", strings.NewReader(body))
		req.Header.Set("x-hc-user-id", user)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Cat-Led-Request", "1")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		var data UserPreferenceResponse
		if w.Code == 200 {
			if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
				t.Fatal(err)
			}
		}
		return w.Code, data
	}
	code, pref := request("GET", "alice", "")
	if code != 200 || pref.BulbStyle != "lava" || !pref.ShowSchedules || !pref.RemindersEnabled || pref.ReminderMinutes != 5 {
		t.Fatalf("migration: %d %+v", code, pref)
	}
	code, pref = request("PUT", "alice", `{"show_schedules":false,"reminders_enabled":false,"reminder_minutes":12,"user_id":"bob"}`)
	if code != 200 || pref.UserID != "alice" || pref.ShowSchedules || pref.RemindersEnabled || pref.ReminderMinutes != 12 || pref.BulbStyle != "lava" {
		t.Fatalf("update: %d %+v", code, pref)
	}
	code, pref = request("PUT", "alice", `{"bulb_style":"classic"}`)
	if code != 200 || pref.ShowSchedules || pref.RemindersEnabled || pref.ReminderMinutes != 12 || pref.BulbStyle != "classic" {
		t.Fatalf("preservation: %d %+v", code, pref)
	}
	_, bob := request("GET", "bob", "")
	if !bob.ShowSchedules || !bob.RemindersEnabled || bob.ReminderMinutes != 5 {
		t.Fatalf("cross-user leak: %+v", bob)
	}
	for _, body := range []string{`{"reminder_minutes":0}`, `{"reminder_minutes":1441}`, `{"reminder_minutes":1.5}`, `{"show_schedules":"false"}`, `{"bulb_style":""}`} {
		if code, _ := request("PUT", "alice", body); code != 400 {
			t.Fatalf("accepted invalid %s: %d", body, code)
		}
	}
	if code, _ := request("GET", "", ""); code != 401 {
		t.Fatalf("anonymous: %d", code)
	}
	// A new browser reads the same saved account settings from the database.
	_, pref = request("GET", "alice", "")
	if pref.ShowSchedules || pref.ReminderMinutes != 12 {
		t.Fatalf("readback: %+v", pref)
	}
}
