package biz

import (
	"context"
	"errors"
	"fmt"

	"cat-led/internal/ent"
	"cat-led/internal/ent/schedule"
	"cat-led/internal/pkg/zlog"

	"github.com/google/uuid"
	_ "github.com/lib-x/entsqlite"
)

// Error messages for schedule operations.
var (
	ErrCreatorRequired      = errors.New("creator is required")
	ErrPermissionDenied     = errors.New("you don't have permission to edit this schedule")
	ErrOnlyCreatorCanDelete = errors.New("only the creator can delete this schedule")
)

// Default templates for ServerChan notifications.
const (
	defaultOnTemplate  = "{{.Name}} 任务执行成功，灯已开启"
	defaultOffTemplate = "{{.Name}} 任务执行成功，灯已关闭"
)

// ScheduleUsecase provides business logic for schedule management.
type ScheduleUsecase struct {
	client *ent.Client
	logger *zlog.Logger
}

// NewScheduleUseCase creates a new ScheduleUsecase with the given database path.
func NewScheduleUseCase(dbPath string, logger *zlog.Logger) *ScheduleUsecase {
	dataSourceName := fmt.Sprintf(
		"file:%s?cache=shared&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(10000)",
		dbPath,
	)

	entClient, err := ent.Open("sqlite3", dataSourceName)
	if err != nil {
		logger.Error().Err(err).Msg("无法连接到数据库")
		return nil
	}

	ctx := context.Background()
	if err := entClient.Schema.Create(ctx); err != nil {
		logger.Error().Err(err).Msg("无法创建数据库表")
		entClient.Close()
		return nil
	}

	logger.Info().Msg("数据库初始化成功，已创建必要的表结构")
	return &ScheduleUsecase{
		client: entClient,
		logger: logger,
	}
}

// GetClient returns the underlying database client.
func (s *ScheduleUsecase) GetClient() *ent.Client {
	return s.client
}

// CreateSchedule creates a new schedule with the given parameters.
func (s *ScheduleUsecase) CreateSchedule(ctx context.Context, sched *ent.Schedule) (*ent.Schedule, error) {
	if sched.Creator == "" {
		return nil, ErrCreatorRequired
	}

	result, err := s.client.Schedule.Create().
		SetName(sched.Name).
		SetCreator(sched.Creator).
		SetWeekDays(sched.WeekDays).
		SetHour(sched.Hour).
		SetMinute(sched.Minute).
		SetOperation(sched.Operation).
		SetEnabled(sched.Enabled).
		SetAllowEditByOthers(sched.AllowEditByOthers).
		SetNotifyViaServerChan(sched.NotifyViaServerChan).
		SetNotifyViaLzc(sched.NotifyViaLzc).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create schedule: %w", err)
	}

	return result, nil
}

// GetSchedule retrieves a schedule by ID.
func (s *ScheduleUsecase) GetSchedule(ctx context.Context, id uuid.UUID) (*ent.Schedule, error) {
	result, err := s.client.Schedule.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}
	return result, nil
}

// GetAllSchedules retrieves all schedules.
func (s *ScheduleUsecase) GetAllSchedules(ctx context.Context) ([]*ent.Schedule, error) {
	result, err := s.client.Schedule.Query().All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all schedules: %w", err)
	}
	return result, nil
}

// GetSchedulesByCreator retrieves all schedules created by a specific user.
func (s *ScheduleUsecase) GetSchedulesByCreator(ctx context.Context, creator string) ([]*ent.Schedule, error) {
	result, err := s.client.Schedule.Query().
		Where(schedule.Creator(creator)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get schedules by creator: %w", err)
	}
	return result, nil
}

// UpdateSchedule updates an existing schedule if the user has permission.
func (s *ScheduleUsecase) UpdateSchedule(ctx context.Context, sched *ent.Schedule, currentUser string) (*ent.Schedule, error) {
	existing, err := s.GetSchedule(ctx, sched.ID)
	if err != nil {
		return nil, err
	}

	if !s.canEdit(existing, currentUser) {
		return nil, ErrPermissionDenied
	}

	updateQuery := s.client.Schedule.UpdateOneID(sched.ID).
		SetName(sched.Name).
		SetWeekDays(sched.WeekDays).
		SetHour(sched.Hour).
		SetMinute(sched.Minute).
		SetOperation(sched.Operation).
		SetEnabled(sched.Enabled).
		SetNotifyViaServerChan(sched.NotifyViaServerChan).
		SetNotifyViaLzc(sched.NotifyViaLzc)

	// Only the creator can change the AllowEditByOthers setting
	if existing.Creator == currentUser {
		updateQuery = updateQuery.SetAllowEditByOthers(sched.AllowEditByOthers)
	}

	result, err := updateQuery.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update schedule: %w", err)
	}

	return result, nil
}

// SetScheduleEnabled updates a schedule's enabled state from trusted system flows.
func (s *ScheduleUsecase) SetScheduleEnabled(ctx context.Context, id uuid.UUID, enabled bool) error {
	if err := s.client.Schedule.UpdateOneID(id).
		SetEnabled(enabled).
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to set schedule enabled state: %w", err)
	}
	return nil
}

// canEdit checks if the user can edit the schedule.
func (s *ScheduleUsecase) canEdit(sched *ent.Schedule, currentUser string) bool {
	return sched.Creator == currentUser || sched.AllowEditByOthers
}

// DeleteSchedule deletes a schedule by ID if the user is the creator.
func (s *ScheduleUsecase) DeleteSchedule(ctx context.Context, id uuid.UUID, currentUser string) error {
	existing, err := s.GetSchedule(ctx, id)
	if err != nil {
		return err
	}

	if existing.Creator != currentUser {
		return ErrOnlyCreatorCanDelete
	}

	if err := s.client.Schedule.DeleteOneID(id).Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
	}

	return nil
}

// GetServerChanConfig retrieves the ServerChan configuration.
func (s *ScheduleUsecase) GetServerChanConfig(ctx context.Context) (*ent.ServerChanConfig, error) {
	config, err := s.client.ServerChanConfig.Query().First(ctx)
	if err != nil {
		return s.defaultServerChanConfig(), nil
	}
	return config, nil
}

// defaultServerChanConfig returns the default ServerChan configuration.
func (s *ScheduleUsecase) defaultServerChanConfig() *ent.ServerChanConfig {
	return &ent.ServerChanConfig{
		SendKey:     "",
		OnTemplate:  defaultOnTemplate,
		OffTemplate: defaultOffTemplate,
		Enabled:     false,
	}
}

// SaveServerChanConfig saves the ServerChan configuration (creates or updates).
func (s *ScheduleUsecase) SaveServerChanConfig(ctx context.Context, config *ent.ServerChanConfig) error {
	existing, err := s.client.ServerChanConfig.Query().First(ctx)
	if err != nil {
		return s.createServerChanConfig(ctx, config)
	}
	return s.updateServerChanConfig(ctx, existing.ID, config)
}

// createServerChanConfig creates a new ServerChan configuration.
func (s *ScheduleUsecase) createServerChanConfig(ctx context.Context, config *ent.ServerChanConfig) error {
	_, err := s.client.ServerChanConfig.Create().
		SetEnabled(config.Enabled).
		SetSendKey(config.SendKey).
		SetOnTemplate(config.OnTemplate).
		SetOffTemplate(config.OffTemplate).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to create serverchan config: %w", err)
	}
	return nil
}

// updateServerChanConfig updates an existing ServerChan configuration.
func (s *ScheduleUsecase) updateServerChanConfig(ctx context.Context, id int, config *ent.ServerChanConfig) error {
	_, err := s.client.ServerChanConfig.UpdateOneID(id).
		SetEnabled(config.Enabled).
		SetSendKey(config.SendKey).
		SetOnTemplate(config.OnTemplate).
		SetOffTemplate(config.OffTemplate).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to update serverchan config: %w", err)
	}
	return nil
}

// Close closes the database client.
func (s *ScheduleUsecase) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}
