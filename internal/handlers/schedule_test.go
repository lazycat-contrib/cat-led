package handlers

import (
	"context"
	"io"
	"path/filepath"
	"testing"

	"cat-led/internal/biz"
	"cat-led/internal/ent"
	"cat-led/internal/ent/schedule"
	"cat-led/internal/pkg/zlog"

	"github.com/rs/zerolog"
)

func TestShouldRunOnWeekdayAllowsOneTimeSchedules(t *testing.T) {
	if !shouldRunOnWeekday(nil, 2) {
		t.Fatal("nil weekdays should represent a one-time schedule and be runnable")
	}
	if !shouldRunOnWeekday([]int{}, 2) {
		t.Fatal("empty weekdays should represent a one-time schedule and be runnable")
	}
	if !shouldRunOnWeekday([]int{2}, 2) {
		t.Fatal("matching repeated weekday should be runnable")
	}
	if shouldRunOnWeekday([]int{1, 3}, 2) {
		t.Fatal("non-matching repeated weekday should not be runnable")
	}
}

func TestDisableOneTimeScheduleAfterRunDisablesOnlyOneTimeSchedules(t *testing.T) {
	ctx := context.Background()
	logger := newDiscardLogger()
	uc := newTestScheduleUsecase(t, logger)

	previousScheduleUseCase := scheduleUseCase
	scheduleUseCase = uc
	t.Cleanup(func() {
		scheduleUseCase = previousScheduleUseCase
	})

	oneTime := createTestSchedule(t, ctx, uc, "one-time", []int{})
	repeated := createTestSchedule(t, ctx, uc, "repeated", []int{2})

	disableOneTimeScheduleAfterRun(ctx, logger, oneTime)
	disableOneTimeScheduleAfterRun(ctx, logger, repeated)

	reloadedOneTime, err := uc.GetSchedule(ctx, oneTime.ID)
	if err != nil {
		t.Fatalf("reload one-time schedule: %v", err)
	}
	if reloadedOneTime.Enabled {
		t.Fatal("one-time schedule should be disabled after running")
	}

	reloadedRepeated, err := uc.GetSchedule(ctx, repeated.ID)
	if err != nil {
		t.Fatalf("reload repeated schedule: %v", err)
	}
	if !reloadedRepeated.Enabled {
		t.Fatal("repeated schedule should remain enabled after running")
	}
}

func newTestScheduleUsecase(t *testing.T, logger *zlog.Logger) *biz.ScheduleUsecase {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "schedules.db")
	uc := biz.NewScheduleUseCase(dbPath, logger)
	if uc == nil {
		t.Fatal("expected schedule usecase to initialize")
	}
	t.Cleanup(func() {
		if err := uc.Close(); err != nil {
			t.Fatalf("close schedule usecase: %v", err)
		}
	})
	return uc
}

func createTestSchedule(t *testing.T, ctx context.Context, uc *biz.ScheduleUsecase, name string, weekDays []int) *ent.Schedule {
	t.Helper()

	s, err := uc.CreateSchedule(ctx, &ent.Schedule{
		Name:      name,
		Creator:   "test-user",
		WeekDays:  weekDays,
		Hour:      17,
		Minute:    25,
		Operation: schedule.OperationOff,
		Enabled:   true,
	})
	if err != nil {
		t.Fatalf("create schedule %q: %v", name, err)
	}
	return s
}

func newDiscardLogger() *zlog.Logger {
	logger := zerolog.New(io.Discard)
	return &zlog.Logger{Logger: logger}
}
