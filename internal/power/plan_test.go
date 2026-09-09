package power

import (
	"encoding/json"
	"testing"
	"time"
)

func TestLegacyPlanRemainsPaired(t *testing.T) {
	var spec Spec
	if err := json.Unmarshal([]byte(`{"mode":"weekly","timezone":"UTC","weekdays":[1],"shutdown_time":"23:00","wake_time":"07:00"}`), &spec); err != nil {
		t.Fatal(err)
	}
	if !spec.HasShutdown() || !spec.HasWake() {
		t.Fatal("legacy operations were disabled")
	}
	off, on, err := spec.Next(time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC))
	if err != nil || on.Sub(off) != 8*time.Hour {
		t.Fatalf("%v %v %v", off, on, err)
	}
}
func TestDisabledTimesAreIgnored(t *testing.T) {
	now := time.Date(2026, 9, 9, 0, 0, 0, 0, time.UTC)
	for _, spec := range []Spec{
		{Mode: "once", Timezone: "UTC", ShutdownEnabled: boolPointer(false), WakeAt: now.Add(time.Hour)},
		{Mode: "once", Timezone: "UTC", WakeEnabled: boolPointer(false), ShutdownAt: now.Add(time.Hour)},
		{Mode: "weekly", Timezone: "UTC", Weekdays: []int{3}, ShutdownEnabled: boolPointer(false), ShutdownTime: "invalid", WakeTime: "01:00"},
		{Mode: "weekly", Timezone: "UTC", Weekdays: []int{3}, WakeEnabled: boolPointer(false), WakeTime: "invalid", ShutdownTime: "01:00"},
	} {
		off, on, err := spec.Next(now)
		if err != nil {
			t.Fatal(err)
		}
		if !spec.HasShutdown() && !off.IsZero() {
			t.Fatal("disabled shutdown retained target")
		}
		if !spec.HasWake() && !on.IsZero() {
			t.Fatal("disabled wake retained target")
		}
	}
}
func TestWakeOnlyRejectsPastOrMissingTarget(t *testing.T) {
	now := time.Now()
	for _, target := range []time.Time{{}, now.Add(-time.Hour)} {
		spec := Spec{Mode: "once", Timezone: "UTC", ShutdownEnabled: boolPointer(false), WakeAt: target}
		if _, _, err := spec.Next(now); err == nil {
			t.Fatal("invalid wake target accepted")
		}
	}
}
