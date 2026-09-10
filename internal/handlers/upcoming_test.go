package handlers

import (
	"cat-led/internal/ent"
	"testing"
	"time"
)

func TestNextScheduleTime(t *testing.T) {
	zone := time.FixedZone("device", 8*3600)
	now := time.Date(2026, 9, 10, 23, 58, 0, 0, zone)
	for _, tc := range []struct {
		name         string
		enabled      bool
		days         []int
		hour, minute int
		want         string
	}{
		{"one time crosses midnight", true, nil, 0, 2, "2026-09-11T00:02:00+08:00"},
		{"weekday skips tomorrow", true, []int{6}, 0, 2, "2026-09-12T00:02:00+08:00"},
		{"next week", true, []int{4}, 23, 57, "2026-09-17T23:57:00+08:00"},
		{"disabled", false, nil, 0, 2, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := nextScheduleTime(&ent.Schedule{Enabled: tc.enabled, WeekDays: tc.days, Hour: tc.hour, Minute: tc.minute}, now)
			if tc.want == "" {
				if !got.IsZero() {
					t.Fatal(got)
				}
				return
			}
			if got.Format(time.RFC3339) != tc.want {
				t.Fatalf("got %s want %s", got, tc.want)
			}
		})
	}
}
func TestUpcomingVisibilityAndOrdering(t *testing.T) {
	now := time.Date(2026, 9, 10, 8, 0, 0, 0, time.UTC)
	all := []*ent.Schedule{
		{Name: "own", Creator: "alice", Enabled: true, Hour: 10},
		{Name: "private", Creator: "bob", Enabled: true, Hour: 9},
		{Name: "shared", Creator: "bob", Enabled: true, AllowEditByOthers: true, Hour: 9},
		{Name: "disabled", Creator: "alice", Hour: 9},
	}
	got := upcomingSchedules(all, "alice", now)
	if len(got) != 2 || got[0].Name != "shared" || got[1].Name != "own" {
		t.Fatalf("%+v", got)
	}
}

func TestNextScheduleTimeDST(t *testing.T) {
	zone, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 1, 5, 40, 0, 0, time.UTC).In(zone)
	got := nextScheduleTime(&ent.Schedule{Enabled: true, Hour: 1, Minute: 30, WeekDays: []int{0}}, now)
	if got.Sub(now) != 50*time.Minute {
		t.Fatalf("fallback next: %s", got)
	}
	now = time.Date(2026, 3, 8, 0, 0, 0, 0, zone)
	got = nextScheduleTime(&ent.Schedule{Enabled: true, Hour: 2, Minute: 30}, now)
	if got.Day() != 9 || got.Hour() != 2 {
		t.Fatalf("nonexistent local time: %s", got)
	}
}
