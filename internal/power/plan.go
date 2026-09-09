package power

import (
	"errors"
	"fmt"
	"slices"
	"time"
	_ "time/tzdata"
)

// Spec describes a device-wide shutdown/wake pair. Weekdays use Sunday=0.
type Spec struct {
	Mode         string    `json:"mode"`
	Timezone     string    `json:"timezone"`
	Weekdays     []int     `json:"weekdays"`
	ShutdownTime string    `json:"shutdown_time"`
	WakeTime     string    `json:"wake_time"`
	ShutdownAt   time.Time `json:"shutdown_at"`
	WakeAt       time.Time `json:"wake_at"`
}

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
	loc, err := time.LoadLocation(s.Timezone)
	if err != nil || s.Timezone == "" {
		return time.Time{}, time.Time{}, errors.New("请选择有效的时区")
	}
	if s.Mode == "once" {
		if !s.ShutdownAt.After(after) {
			return time.Time{}, time.Time{}, errors.New("关机时间至少需在两分钟之后")
		}
		if s.WakeAt.Sub(s.ShutdownAt) < 5*time.Minute {
			return time.Time{}, time.Time{}, errors.New("开机时间需比关机时间至少晚五分钟")
		}
		if s.WakeAt.Sub(after) > 28*24*time.Hour {
			return time.Time{}, time.Time{}, errors.New("单次开机时间需在未来 28 天内")
		}
		return s.ShutdownAt.UTC().Truncate(time.Second), s.WakeAt.UTC().Truncate(time.Second), nil
	}
	if s.Mode != "weekly" {
		return time.Time{}, time.Time{}, errors.New("请选择单次或每周重复")
	}
	if len(s.Weekdays) == 0 || len(s.Weekdays) > 7 {
		return time.Time{}, time.Time{}, errors.New("请选择重复的星期")
	}
	for _, day := range s.Weekdays {
		if day < 0 || day > 6 {
			return time.Time{}, time.Time{}, errors.New("无效的星期")
		}
	}
	off, err := time.Parse("15:04", s.ShutdownTime)
	if err != nil {
		return time.Time{}, time.Time{}, errors.New("关机时间格式应为 HH:mm")
	}
	on, err := time.Parse("15:04", s.WakeTime)
	if err != nil {
		return time.Time{}, time.Time{}, errors.New("开机时间格式应为 HH:mm")
	}
	date := after.In(loc)
	for i := range 16 {
		d := date.AddDate(0, 0, i)
		if !slices.Contains(s.Weekdays, int(d.Weekday())) {
			continue
		}
		shutdown := time.Date(d.Year(), d.Month(), d.Day(), off.Hour(), off.Minute(), 0, 0, loc)
		// Skip nonexistent local times at daylight-saving transitions.
		if shutdown.Format("15:04") != s.ShutdownTime || !shutdown.After(after) {
			continue
		}
		wakeDate := d
		if s.WakeTime <= s.ShutdownTime {
			wakeDate = d.AddDate(0, 0, 1)
		}
		wake := time.Date(wakeDate.Year(), wakeDate.Month(), wakeDate.Day(), on.Hour(), on.Minute(), 0, 0, loc)
		if wake.Format("15:04") != s.WakeTime {
			continue
		}
		if wake.Sub(shutdown) < 5*time.Minute {
			return time.Time{}, time.Time{}, errors.New("开机时间需比关机时间至少晚五分钟")
		}
		return shutdown.UTC(), wake.UTC(), nil
	}
	return time.Time{}, time.Time{}, fmt.Errorf("无法计算下一次计划，请检查时区和时间")
}
