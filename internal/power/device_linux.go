package power

import (
	"errors"
	"fmt"
	"github.com/lib-x/rtc"
	"time"
)

type Device interface {
	Probe() error
	Arm(time.Time, time.Time) error
	Clear(...time.Time) error
}

type LinuxRTC struct{ Path string }

func (d LinuxRTC) Probe() error {
	r, err := rtc.Open(d.Path)
	if err != nil {
		return fmt.Errorf("无法打开 RTC 设备: %w", err)
	}
	defer r.Close()
	now, err := r.Time()
	if err != nil {
		return err
	}
	drift := time.Since(now)
	if drift > 2*time.Minute || drift < -2*time.Minute {
		return errors.New("RTC 时钟与系统 UTC 时间相差超过两分钟，请先在系统中校准硬件时钟")
	}
	_, err = r.WakeAlarm()
	return err
}
func (d LinuxRTC) Arm(target, previous time.Time) error {
	r, err := rtc.Open(d.Path)
	if err != nil {
		return err
	}
	defer r.Close()
	now, err := r.Time()
	if err != nil {
		return err
	}
	if delta := time.Since(now); delta > 2*time.Minute || delta < -2*time.Minute {
		return errors.New("RTC 时钟与系统 UTC 时间不一致")
	}
	alarm, err := r.WakeAlarm()
	if err != nil {
		return err
	}
	if alarm.Enabled && alarm.Time.After(now) && !alarm.Time.Equal(previous) {
		return errors.New("RTC 已有其他程序设置的开机闹钟，请先处理该闹钟")
	}
	if err = r.SetWakeAlarm(target); err != nil {
		return err
	}
	alarm, err = r.WakeAlarm()
	if err != nil {
		return err
	}
	if !alarm.Enabled || !alarm.Time.Equal(target) {
		return errors.New("RTC 闹钟回读与计划不一致，已阻止关机")
	}
	return nil
}
func (d LinuxRTC) Clear(expected ...time.Time) error {
	hasExpected := false
	for _, target := range expected {
		if !target.IsZero() {
			hasExpected = true
		}
	}
	if !hasExpected {
		return nil
	}
	r, err := rtc.Open(d.Path)
	if err != nil {
		return err
	}
	defer r.Close()
	alarm, err := r.WakeAlarm()
	if err != nil {
		return err
	}
	if !alarm.Enabled {
		return nil
	}
	owned := false
	for _, target := range expected {
		if alarm.Time.Equal(target) {
			owned = true
		}
	}
	if !owned {
		return errors.New("RTC 闹钟已被其他程序修改，未清除其他程序的闹钟")
	}
	if err = r.CancelWakeAlarm(); err != nil {
		return err
	}
	alarm, err = r.WakeAlarm()
	if err != nil {
		return err
	}
	if alarm.Enabled {
		return errors.New("RTC 闹钟取消后仍启用")
	}
	return nil
}
