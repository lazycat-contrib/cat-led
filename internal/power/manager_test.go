package power

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

type memoryStore struct {
	state         State
	saveErr       error
	failAt, saves int
}

func (s *memoryStore) Load(context.Context) (State, error) { return s.state, nil }
func (s *memoryStore) Save(_ context.Context, state State) error {
	s.saves++
	if s.saveErr != nil || (s.failAt > 0 && s.saves == s.failAt) {
		return errors.New("disk failure")
	}
	s.state = state
	return nil
}

type fakeRTC struct {
	target           time.Time
	armErr, clearErr error
	arms, clears     int
	probes           int
	probeErr         error
}

func (d *fakeRTC) Probe() error { d.probes++; return d.probeErr }
func (d *fakeRTC) Arm(target, previous time.Time) error {
	d.arms++
	if d.armErr != nil {
		return d.armErr
	}
	if !d.target.IsZero() && !d.target.Equal(previous) {
		return errors.New("foreign alarm")
	}
	d.target = target
	return nil
}
func (d *fakeRTC) Clear(expected ...time.Time) error {
	d.clears++
	if d.clearErr != nil {
		return d.clearErr
	}
	if d.target.IsZero() {
		return nil
	}
	for _, target := range expected {
		if d.target.Equal(target) {
			d.target = time.Time{}
			return nil
		}
	}
	return errors.New("alarm is not owned by this plan")
}

func fixture(t *testing.T) (*Manager, *memoryStore, *fakeRTC, *int, time.Time, Spec) {
	t.Helper()
	now := time.Date(2026, 9, 9, 0, 0, 0, 0, time.UTC)
	spec := Spec{Mode: "once", Timezone: "Asia/Shanghai", ShutdownAt: now.Add(time.Hour), WakeAt: now.Add(9 * time.Hour)}
	store := &memoryStore{state: State{Phase: "disabled"}}
	device := &fakeRTC{}
	calls := new(int)
	m := New(store, device, func(context.Context, string) (bool, error) { return true, nil }, func(context.Context) error { *calls++; return nil })
	return m, store, device, calls, now, spec
}
func TestShutdownOnceAndRecovery(t *testing.T) {
	m, store, _, calls, now, spec := fixture(t)
	if _, err := m.Save(t.Context(), spec, "admin", now); err != nil {
		t.Fatal(err)
	}
	if err := m.Step(t.Context(), spec.ShutdownAt, false); err != nil {
		t.Fatal(err)
	}
	if *calls != 1 || store.state.Phase != "waiting_wake" {
		t.Fatal("shutdown not recorded")
	}
	if err := m.Step(t.Context(), spec.ShutdownAt.Add(time.Second), true); err != nil {
		t.Fatal(err)
	}
	if *calls != 1 {
		t.Fatal("shutdown replayed after restart")
	}
	if err := m.Step(t.Context(), spec.WakeAt, true); err != nil {
		t.Fatal(err)
	}
	if store.state.Phase != "completed" {
		t.Fatal(store.state.Phase)
	}
}
func TestMissedShutdownNeverCatchesUp(t *testing.T) {
	for _, recovery := range []bool{true, false} {
		t.Run(map[bool]string{true: "restart", false: "late_tick"}[recovery], func(t *testing.T) {
			m, store, _, calls, now, spec := fixture(t)
			_, _ = m.Save(t.Context(), spec, "admin", now)
			if err := m.Step(t.Context(), spec.ShutdownAt.Add(time.Minute), recovery); err != nil {
				t.Fatal(err)
			}
			if *calls != 0 || store.state.Phase != "missed" {
				t.Fatal("missed task executed")
			}
		})
	}
}
func TestFailuresPreventShutdown(t *testing.T) {
	for _, kind := range []string{"rtc", "storage", "role", "rpc", "preparing"} {
		t.Run(kind, func(t *testing.T) {
			m, store, d, calls, now, spec := fixture(t)
			_, _ = m.Save(t.Context(), spec, "admin", now)
			switch kind {
			case "rtc":
				d.armErr = errors.New("rtc denied")
			case "storage":
				store.saveErr = errors.New("disk full")
			case "role":
				m.admin = func(context.Context, string) (bool, error) { return false, nil }
			case "rpc":
				m.shutdown = func(context.Context) error { *calls++; return errors.New("rpc ambiguous") }
			case "preparing":
				store.state.Phase = "preparing"
			}
			if err := m.Step(t.Context(), spec.ShutdownAt, false); err == nil {
				t.Fatal("failure not reported")
			}
			_ = m.Step(t.Context(), spec.ShutdownAt, true)
			expected := 0
			if kind == "rpc" {
				expected = 1
			}
			if *calls != expected {
				t.Fatalf("shutdown calls %d", *calls)
			}
		})
	}
}
func TestSaveFailureAndCancellation(t *testing.T) {
	m, store, d, calls, now, spec := fixture(t)
	store.failAt = 2
	if _, err := m.Save(t.Context(), spec, "admin", now); err == nil {
		t.Fatal("expected durable activation failure")
	}
	if store.state.Phase != "preparing" || d.clears != 1 {
		t.Fatal("unsafe activation")
	}
	store.failAt = 0
	_, _ = m.Save(t.Context(), spec, "admin", now)
	d.clearErr = errors.New("device offline")
	if _, err := m.Cancel(t.Context(), now); err == nil {
		t.Fatal("cancel should report failure")
	}
	if store.state.Phase != "disabled" {
		t.Fatal("shutdown still enabled")
	}
	_ = m.Step(t.Context(), spec.ShutdownAt, false)
	if *calls != 0 {
		t.Fatal("cancelled task ran")
	}
}
func TestConcurrentTickDoesNotDuplicateShutdown(t *testing.T) {
	m, _, _, calls, now, spec := fixture(t)
	_, _ = m.Save(t.Context(), spec, "admin", now)
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() { _ = m.Step(t.Context(), spec.ShutdownAt, false) })
	}
	wg.Wait()
	if *calls != 1 {
		t.Fatal(*calls)
	}
}
func TestSQLStoreReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "existing.db")
	s, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	want := State{Phase: "waiting_wake", Creator: "admin", WakeAt: time.Now().UTC().Truncate(time.Second)}
	if err = s.Save(t.Context(), want); err != nil {
		t.Fatal(err)
	}
	s.Close()
	s, err = OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	got, err := s.Load(t.Context())
	if err != nil || got.Creator != want.Creator || !got.WakeAt.Equal(want.WakeAt) {
		t.Fatalf("%+v %v", got, err)
	}
}
func TestWeeklyOvernightAndRollover(t *testing.T) {
	now := time.Date(2026, 9, 9, 14, 0, 0, 0, time.UTC)
	spec := Spec{Mode: "weekly", Timezone: "Asia/Shanghai", Weekdays: []int{3}, ShutdownTime: "23:00", WakeTime: "07:00"}
	off, on, err := spec.Next(now)
	if err != nil {
		t.Fatal(err)
	}
	if off.Hour() != 15 || on.Hour() != 23 || on.Sub(off) != 8*time.Hour {
		t.Fatalf("%v %v", off, on)
	}
	next, _, err := spec.Next(off)
	if err != nil || next.Sub(off) != 7*24*time.Hour {
		t.Fatalf("%v %v", next, err)
	}
}
func TestInvalidPlans(t *testing.T) {
	_, _, _, _, now, spec := fixture(t)
	cases := []Spec{{Mode: "once", Timezone: "UTC", ShutdownAt: now}, {Mode: "weekly", Timezone: "UTC"}, {Mode: "weekly", Timezone: "UTC", Weekdays: []int{9}}, {Mode: "weekly", Timezone: "bad"}, {Mode: "bad", Timezone: "UTC"}}
	short := spec
	short.WakeAt = short.ShutdownAt
	cases = append(cases, short)
	for _, s := range cases {
		if _, _, err := s.Next(now); err == nil {
			t.Fatalf("accepted %+v", s)
		}
	}
}

func TestFailedReplacementRetainsPreviousAlarmOwnership(t *testing.T) {
	m, store, d, _, now, spec := fixture(t)
	if _, err := m.Save(t.Context(), spec, "admin", now); err != nil {
		t.Fatal(err)
	}
	previous := store.state.WakeAt
	spec.WakeAt = spec.WakeAt.Add(time.Hour)
	d.armErr = errors.New("write failed")
	if _, err := m.Save(t.Context(), spec, "admin", now); err == nil {
		t.Fatal("expected write failure")
	}
	if !store.state.PreviousWakeAt.Equal(previous) {
		t.Fatal("lost original alarm after failed replacement")
	}
	if store.state.Phase != "error" {
		t.Fatal("failed replacement still active")
	}
}
func TestWeeklyAdvancesAfterWake(t *testing.T) {
	m, store, d, calls, now, _ := fixture(t)
	spec := Spec{Mode: "weekly", Timezone: "Asia/Shanghai", Weekdays: []int{3}, ShutdownTime: "23:00", WakeTime: "07:00"}
	first, err := m.Save(t.Context(), spec, "admin", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Step(t.Context(), first.ShutdownAt, false); err != nil {
		t.Fatal(err)
	}
	if err := m.Step(t.Context(), first.WakeAt, true); err != nil {
		t.Fatal(err)
	}
	if *calls != 1 || store.state.Phase != "active" || store.state.ShutdownAt.Sub(first.ShutdownAt) != 7*24*time.Hour || !d.target.Equal(store.state.WakeAt) {
		t.Fatalf("%+v calls=%d", store.state, *calls)
	}
}

func TestRepeatedFailedReplacementResolvesOriginalAlarm(t *testing.T) {
	m, store, d, _, now, spec := fixture(t)
	_, err := m.Save(t.Context(), spec, "admin", now)
	if err != nil {
		t.Fatal(err)
	}
	spec.WakeAt = spec.WakeAt.Add(time.Hour)
	d.armErr = errors.New("write failed")
	_, _ = m.Save(t.Context(), spec, "admin", now)
	spec.WakeAt = spec.WakeAt.Add(time.Hour)
	_, _ = m.Save(t.Context(), spec, "admin", now)
	if d.clears != 1 {
		t.Fatal("older alarm was not reconciled before second replacement")
	}
	if !store.state.PreviousWakeAt.IsZero() {
		t.Fatal("unresolved older alarm carried forward")
	}
	if _, err := m.Cancel(t.Context(), now); err != nil || !d.target.IsZero() {
		t.Fatalf("old alarm could not be cancelled: %v", err)
	}
}
func TestWeeklySkipsNonexistentDSTOccurrence(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	spec := Spec{Mode: "weekly", Timezone: "America/New_York", Weekdays: []int{0}, ShutdownTime: "02:30", WakeTime: "07:00"}
	off, _, err := spec.Next(time.Date(2026, 3, 2, 0, 0, 0, 0, loc))
	if err != nil || off.In(loc).Day() != 15 {
		t.Fatalf("%v %v", off, err)
	}
}
func TestMinimumGapAllowsNormalTickerDelay(t *testing.T) {
	m, _, _, calls, now, spec := fixture(t)
	spec.WakeAt = spec.ShutdownAt.Add(5 * time.Minute)
	if _, err := m.Save(t.Context(), spec, "admin", now); err != nil {
		t.Fatal(err)
	}
	if err := m.Step(t.Context(), spec.ShutdownAt.Add(10*time.Second), false); err != nil {
		t.Fatal(err)
	}
	if *calls != 1 {
		t.Fatal("valid minimum interval skipped")
	}
}

func boolPointer(value bool) *bool { return &value }

func TestShutdownOnlyDoesNotRequireRTC(t *testing.T) {
	m, store, d, calls, now, spec := fixture(t)
	spec.WakeEnabled = boolPointer(false)
	spec.WakeAt = time.Time{}
	d.probeErr = errors.New("no RTC device")
	d.armErr = d.probeErr
	if _, err := m.Save(t.Context(), spec, "admin", now); err != nil {
		t.Fatal(err)
	}
	if d.probes != 0 || d.arms != 0 || d.clears != 0 {
		t.Fatal("shutdown-only plan accessed RTC")
	}
	if err := m.Step(t.Context(), now, true); err != nil {
		t.Fatal(err)
	}
	if err := m.Step(t.Context(), spec.ShutdownAt, false); err != nil {
		t.Fatal(err)
	}
	if *calls != 1 || store.state.Phase != "shutdown_sent" {
		t.Fatalf("%d %s", *calls, store.state.Phase)
	}
	if err := m.Step(t.Context(), spec.ShutdownAt.Add(time.Second), true); err != nil {
		t.Fatal(err)
	}
	if *calls != 1 || store.state.Phase != "completed" {
		t.Fatal("shutdown replayed after recovery")
	}
}
func TestWakeOnlyNeverShutsDown(t *testing.T) {
	m, store, d, calls, now, spec := fixture(t)
	spec.ShutdownEnabled = boolPointer(false)
	spec.ShutdownAt = time.Time{}
	if _, err := m.Save(t.Context(), spec, "admin", now); err != nil {
		t.Fatal(err)
	}
	if !store.state.ShutdownAt.IsZero() || !d.target.Equal(spec.WakeAt) {
		t.Fatal("incorrect independent wake target")
	}
	if err := m.Step(t.Context(), now, true); err != nil {
		t.Fatal(err)
	}
	if err := m.Step(t.Context(), spec.WakeAt, false); err != nil {
		t.Fatal(err)
	}
	if *calls != 0 || store.state.Phase != "completed" {
		t.Fatal("wake-only triggered shutdown")
	}
}
func TestWeeklyIndependentOperations(t *testing.T) {
	for _, wake := range []bool{false, true} {
		t.Run(map[bool]string{false: "shutdown", true: "wake"}[wake], func(t *testing.T) {
			m, store, d, calls, now, _ := fixture(t)
			spec := Spec{Mode: "weekly", Timezone: "Asia/Shanghai", Weekdays: []int{3}, ShutdownEnabled: boolPointer(!wake), WakeEnabled: boolPointer(wake), ShutdownTime: "23:00", WakeTime: "10:00"}
			first, err := m.Save(t.Context(), spec, "admin", now)
			if err != nil {
				t.Fatal(err)
			}
			target := first.ShutdownAt
			if wake {
				target = first.WakeAt
				if target.Hour() != 2 {
					t.Fatal("wake-only was shifted to the next day")
				}
			}
			if err := m.Step(t.Context(), target, false); err != nil {
				t.Fatal(err)
			}
			if !wake {
				if err := m.Step(t.Context(), target.Add(time.Second), true); err != nil {
					t.Fatal(err)
				}
			}
			next := store.state.ShutdownAt
			if wake {
				next = store.state.WakeAt
			}
			if next.Sub(target) != 7*24*time.Hour || store.state.Phase != "active" {
				t.Fatalf("%+v", store.state)
			}
			if wake && (*calls != 0 || !d.target.Equal(next)) {
				t.Fatal("wake recurrence incorrect")
			}
			if !wake && (*calls != 1 || d.arms != 0) {
				t.Fatal("shutdown recurrence touched RTC or repeated shutdown")
			}
		})
	}
}
func TestRemovingWakeClearsPreviousAlarm(t *testing.T) {
	m, store, d, _, now, spec := fixture(t)
	_, err := m.Save(t.Context(), spec, "admin", now)
	if err != nil {
		t.Fatal(err)
	}
	old := spec.WakeAt
	spec.WakeEnabled = boolPointer(false)
	d.clearErr = errors.New("device unavailable")
	if _, err := m.Save(t.Context(), spec, "admin", now); err == nil {
		t.Fatal("failed removal enabled shutdown")
	}
	if store.state.Phase != "error" || !store.state.PreviousWakeAt.Equal(old) {
		t.Fatal("previous alarm ownership lost")
	}
	d.clearErr = nil
	if _, err := m.Save(t.Context(), spec, "admin", now); err != nil {
		t.Fatal(err)
	}
	if !d.target.IsZero() || !store.state.WakeAt.IsZero() || store.state.Phase != "active" {
		t.Fatal("old RTC alarm still armed")
	}
}
func TestBothDisabledCancelsWithoutTimeFields(t *testing.T) {
	m, store, d, _, now, spec := fixture(t)
	_, err := m.Save(t.Context(), spec, "admin", now)
	if err != nil {
		t.Fatal(err)
	}
	disabled := Spec{ShutdownEnabled: boolPointer(false), WakeEnabled: boolPointer(false)}
	if _, err := m.Save(t.Context(), disabled, "admin", now); err != nil {
		t.Fatal(err)
	}
	if store.state.Phase != "disabled" || !d.target.IsZero() {
		t.Fatal("off switches failed to cancel")
	}
}

func TestWakeRearmChecksCurrentAdmin(t *testing.T) {
	for _, recovery := range []bool{true, false} {
		for _, lookupError := range []bool{true, false} {
			t.Run(fmt.Sprintf("recovery=%t/error=%t", recovery, lookupError), func(t *testing.T) {
				m, store, d, _, now, _ := fixture(t)
				spec := Spec{Mode: "weekly", Timezone: "UTC", Weekdays: []int{3}, ShutdownEnabled: boolPointer(false), WakeTime: "07:00"}
				first, err := m.Save(t.Context(), spec, "admin", now)
				if err != nil {
					t.Fatal(err)
				}
				arms := d.arms
				m.admin = func(context.Context, string) (bool, error) {
					if lookupError {
						return false, errors.New("offline")
					}
					return false, nil
				}
				when := first.WakeAt
				if recovery {
					when = now
				}
				if err := m.Step(t.Context(), when, recovery); err == nil {
					t.Fatal("revoked or unverified administrator accepted")
				}
				if d.arms != arms || store.state.Phase != "error" {
					t.Fatal("unauthorized plan rearmed")
				}
			})
		}
	}
}
