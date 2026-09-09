package power

import (
	"errors"
	"slices"
	"time"
	_ "time/tzdata"
)

// Spec describes independently selectable device power operations. Nil switches
// preserve the paired behavior of plans saved before these fields existed.
// Both operations share the recurrence and timezone; weekdays use Sunday=0.
type Spec struct {
	ShutdownEnabled *bool     `json:"shutdown_enabled,omitempty"`
	WakeEnabled     *bool     `json:"wake_enabled,omitempty"`
	Mode            string    `json:"mode"`
	Timezone        string    `json:"timezone"`
	Weekdays        []int     `json:"weekdays"`
	ShutdownTime    string    `json:"shutdown_time"`
	WakeTime        string    `json:"wake_time"`
	ShutdownAt      time.Time `json:"shutdown_at"`
	WakeAt          time.Time `json:"wake_at"`
}

func (s Spec) HasShutdown() bool { return s.ShutdownEnabled == nil || *s.ShutdownEnabled }
func (s Spec) HasWake() bool     { return s.WakeEnabled == nil || *s.WakeEnabled }

type State struct {
	PreviousWakeAt time.Time `json:"previous_wake_at,omitzero"`
	Spec           Spec      `json:"spec"`
	Phase          string    `json:"phase"`
	ShutdownAt     time.Time `json:"shutdown_at"`
	WakeAt         time.Time `json:"wake_at"`
	Creator        string    `json:"creator"`
	Error          string    `json:"error"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (s Spec) Next(after time.Time) (time.Time, time.Time, error) {
	var shutdown, wake time.Time
	if !s.HasShutdown() && !s.HasWake() {
		return shutdown, wake, nil
	}
	loc, err := time.LoadLocation(s.Timezone)
	if err != nil || s.Timezone == "" {
		return shutdown, wake, errors.New("请选择有效的时区")
	}
	if s.Mode == "once" {
		if s.HasShutdown() {
			shutdown = s.ShutdownAt.UTC().Truncate(time.Second)
			if !shutdown.After(after) {
				return time.Time{}, time.Time{}, errors.New("关机时间至少需在两分钟之后")
			}
		}
		if s.HasWake() {
			wake = s.WakeAt.UTC().Truncate(time.Second)
			if !wake.After(after) {
				return time.Time{}, time.Time{}, errors.New("开机时间至少需在两分钟之后")
			}
			if wake.Sub(after) > 28*24*time.Hour {
				return time.Time{}, time.Time{}, errors.New("单次开机时间需在未来 28 天内")
			}
		}
		if s.HasShutdown() && s.HasWake() && wake.Sub(shutdown) < 5*time.Minute {
			return time.Time{}, time.Time{}, errors.New("同时启用时，开机时间需比关机时间至少晚五分钟")
		}
		return shutdown, wake, nil
	}
	if s.Mode != "weekly" {
		return shutdown, wake, errors.New("请选择单次或每周重复")
	}
	if len(s.Weekdays) == 0 || len(s.Weekdays) > 7 {
		return shutdown, wake, errors.New("请选择重复的星期")
	}
	for _, day := range s.Weekdays {
		if day < 0 || day > 6 {
			return shutdown, wake, errors.New("无效的星期")
		}
	}
	var off, on time.Time
	if s.HasShutdown() {
		off, err = time.Parse("15:04", s.ShutdownTime)
		if err != nil {
			return shutdown, wake, errors.New("关机时间格式应为 HH:mm")
		}
	}
	if s.HasWake() {
		on, err = time.Parse("15:04", s.WakeTime)
		if err != nil {
			return shutdown, wake, errors.New("开机时间格式应为 HH:mm")
		}
	}
	for i := range 16 {
		date := after.In(loc).AddDate(0, 0, i)
		if !slices.Contains(s.Weekdays, int(date.Weekday())) {
			continue
		}
		if s.HasShutdown() {
			shutdown = time.Date(date.Year(), date.Month(), date.Day(), off.Hour(), off.Minute(), 0, 0, loc)
			if shutdown.Format("15:04") != s.ShutdownTime || !shutdown.After(after) {
				continue
			}
		}
		if s.HasWake() {
			wakeDate := date
			// Only paired schedules interpret an earlier wake time as the next day.
			if s.HasShutdown() && s.WakeTime <= s.ShutdownTime {
				wakeDate = date.AddDate(0, 0, 1)
			}
			wake = time.Date(wakeDate.Year(), wakeDate.Month(), wakeDate.Day(), on.Hour(), on.Minute(), 0, 0, loc)
			if wake.Format("15:04") != s.WakeTime || !wake.After(after) {
				continue
			}
			if s.HasShutdown() && wake.Sub(shutdown) < 5*time.Minute {
				return time.Time{}, time.Time{}, errors.New("同时启用时，开机时间需比关机时间至少晚五分钟")
			}
		}
		return shutdown.UTC(), wake.UTC(), nil
	}
	return time.Time{}, time.Time{}, errors.New("无法计算下一次计划，请检查时区和时间")
}
