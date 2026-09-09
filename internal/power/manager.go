package power

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"
)

// Manager serializes the database/hardware boundary. Preparing and waiting_wake
// states are committed before their external effects; recovery never replays a
// possibly completed shutdown. Hardware and SQLite cannot share a transaction.
type Manager struct {
	mu       sync.Mutex
	store    Store
	device   Device
	admin    func(context.Context, string) (bool, error)
	shutdown func(context.Context) error
}

func New(store Store, device Device, admin func(context.Context, string) (bool, error), shutdown func(context.Context) error) *Manager {
	return &Manager{store: store, device: device, admin: admin, shutdown: shutdown}
}

type Status struct {
	State       State     `json:"state"`
	Available   bool      `json:"available"`
	DeviceError string    `json:"device_error"`
	ServerTime  time.Time `json:"server_time"`
}

func (m *Manager) Status(ctx context.Context) (Status, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, err := m.store.Load(ctx)
	if err != nil {
		return Status{}, err
	}
	result := Status{State: state, ServerTime: time.Now().UTC(), Available: true}
	if err := m.device.Probe(); err != nil {
		result.Available = false
		result.DeviceError = err.Error()
	}
	return result, nil
}
func (m *Manager) Save(ctx context.Context, spec Spec, creator string, now time.Time) (State, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	off, on, err := spec.Next(now.Add(2 * time.Minute))
	if err != nil {
		return State{}, err
	}
	old, err := m.store.Load(ctx)
	if err != nil {
		return State{}, err
	}
	// Resolve uncertain ownership before replacing a failed plan again. This
	// prevents successive failures from forgetting an older, still-armed alarm.
	if !old.PreviousWakeAt.IsZero() {
		old.Phase = "disabled"
		if err = m.store.Save(ctx, old); err != nil {
			return old, err
		}
		if err = m.device.Clear(old.WakeAt, old.PreviousWakeAt); err != nil {
			return old, err
		}
		old.WakeAt = time.Time{}
		old.PreviousWakeAt = time.Time{}
	}
	if err = m.device.Probe(); err != nil {
		return old, err
	}
	state := State{Spec: spec, Phase: "preparing", PreviousWakeAt: old.WakeAt, ShutdownAt: off, WakeAt: on, Creator: creator, UpdatedAt: now}
	if err = m.store.Save(ctx, state); err != nil {
		return old, err
	}
	if err = m.device.Arm(on, old.WakeAt); err != nil {
		return state, m.fail(ctx, state, err)
	}
	state.Phase = "active"
	state.PreviousWakeAt = time.Time{}
	if err = m.store.Save(ctx, state); err != nil {
		// Preparing is durable and cannot trigger shutdown after restart.
		return state, errors.Join(err, m.device.Clear(on))
	}
	return state, nil
}
func (m *Manager) Cancel(ctx context.Context, now time.Time) (State, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, err := m.store.Load(ctx)
	if err != nil {
		return state, err
	}
	state.Phase = "disabled"
	state.Error = ""
	state.UpdatedAt = now
	if err = m.store.Save(ctx, state); err != nil {
		return state, err
	}
	// Keep WakeAt on failure so cancellation can be retried.
	if err = m.device.Clear(state.WakeAt, state.PreviousWakeAt); err != nil {
		state.Error = err.Error()
		return state, errors.Join(err, m.store.Save(ctx, state))
	}
	return state, nil
}
func (m *Manager) fail(ctx context.Context, state State, cause error) error {
	state.Phase = "error"
	state.Error = cause.Error()
	state.UpdatedAt = time.Now().UTC()
	return errors.Join(cause, m.store.Save(ctx, state))
}

func (m *Manager) Run(ctx context.Context) {
	if err := m.Step(ctx, time.Now(), true); err != nil {
		log.Printf("RTC recovery: %v", err)
	}
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			if err := m.Step(ctx, now, false); err != nil {
				log.Printf("RTC schedule: %v", err)
			}
		}
	}
}

func (m *Manager) Step(ctx context.Context, now time.Time, recovering bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, err := m.store.Load(ctx)
	if err != nil {
		return err
	}
	switch state.Phase {
	case "preparing":
		return m.fail(ctx, state, errors.New("上次设置未完成，请重新保存计划"))
	case "active":
		if now.Before(state.ShutdownAt) {
			if recovering {
				if err = m.device.Arm(state.WakeAt, state.WakeAt); err != nil {
					return m.fail(ctx, state, err)
				}
			}
			return nil
		}
		if recovering || now.Sub(state.ShutdownAt) > 45*time.Second {
			return m.advance(ctx, state, now, "已跳过错过的关机时间")
		}
		allowed, err := m.admin(ctx, state.Creator)
		if err != nil {
			return m.fail(ctx, state, errors.New("无法核实计划创建者的懒猫管理员权限，已停止计划"))
		}
		if !allowed {
			return m.fail(ctx, state, errors.New("计划创建者已不是懒猫管理员，已停止计划"))
		}
		if state.WakeAt.Sub(now) < time.Minute {
			return m.fail(ctx, state, errors.New("距离开机时间不足一分钟，已阻止关机"))
		}
		if err = m.device.Arm(state.WakeAt, state.WakeAt); err != nil {
			return m.fail(ctx, state, err)
		}
		state.Phase = "waiting_wake"
		state.UpdatedAt = now
		if err = m.store.Save(ctx, state); err != nil {
			return err
		}
		// Once persisted, never retry an ambiguous shutdown response.
		if err = m.shutdown(ctx); err != nil {
			return m.fail(ctx, state, err)
		}
	case "waiting_wake":
		if !now.Before(state.WakeAt) {
			return m.advance(ctx, state, now, "")
		}
	}
	return nil
}
func (m *Manager) advance(ctx context.Context, state State, now time.Time, message string) error {
	if state.Spec.Mode == "once" {
		state.Phase = "completed"
		if message != "" {
			state.Phase = "missed"
		}
		state.Error = message
		state.UpdatedAt = now
		return m.store.Save(ctx, state)
	}
	off, on, err := state.Spec.Next(now.Add(2 * time.Minute))
	if err != nil {
		return m.fail(ctx, state, err)
	}
	previous := state.WakeAt
	state.PreviousWakeAt = previous
	state.ShutdownAt = off
	state.WakeAt = on
	state.Phase = "preparing"
	state.Error = message
	state.UpdatedAt = now
	if err = m.store.Save(ctx, state); err != nil {
		return err
	}
	if err = m.device.Arm(on, previous); err != nil {
		return m.fail(ctx, state, err)
	}
	state.Phase = "active"
	state.PreviousWakeAt = time.Time{}
	return m.store.Save(ctx, state)
}
