package biz

import (
	"cat-led/internal/ent"
	"cat-led/internal/ent/schedule"
	"cat-led/internal/pkg/zlog"
	"context"
	"fmt"

	"github.com/google/uuid"
	_ "github.com/lib-x/entsqlite"
)

type ScheduleUsecase struct {
	client *ent.Client
	logger *zlog.Logger
}

func NewScheduleUseCase(dbPath string, logger *zlog.Logger) *ScheduleUsecase {
	dataSourceName := fmt.Sprintf("file:%s?cache=shared&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(10000)", dbPath)
	entClient, err := ent.Open("sqlite3", dataSourceName)
	if err != nil {
		logger.Error().Err(err).Msg("无法连接到数据库")
		return nil
	}

	// 自动创建数据库表
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

// GetClient 返回数据库客户端
func (s *ScheduleUsecase) GetClient() *ent.Client {
	return s.client
}

// CreateSchedule creates a new schedule with the given parameters
func (s *ScheduleUsecase) CreateSchedule(ctx context.Context, schedule *ent.Schedule) (*ent.Schedule, error) {
	creator := schedule.Creator
	if creator == "" {
		return nil, fmt.Errorf("creator is required")
	}

	result, err := s.client.Schedule.Create().
		SetName(schedule.Name).
		SetCreator(creator).
		SetWeekDays(schedule.WeekDays).
		SetHour(schedule.Hour).
		SetMinute(schedule.Minute).
		SetOperation(schedule.Operation).
		SetEnabled(schedule.Enabled).
		SetAllowEditByOthers(schedule.AllowEditByOthers).
		SetNotifyViaServerChan(schedule.NotifyViaServerChan).
		Save(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to create schedule: %w", err)
	}

	return result, nil
}

// GetSchedule retrieves a schedule by ID
func (s *ScheduleUsecase) GetSchedule(ctx context.Context, id uuid.UUID) (*ent.Schedule, error) {
	result, err := s.client.Schedule.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}
	return result, nil
}

// GetAllSchedules retrieves all schedules
func (s *ScheduleUsecase) GetAllSchedules(ctx context.Context) ([]*ent.Schedule, error) {
	result, err := s.client.Schedule.Query().All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all schedules: %w", err)
	}
	return result, nil
}

// GetSchedulesByCreator retrieves all schedules created by a specific user
func (s *ScheduleUsecase) GetSchedulesByCreator(ctx context.Context, creator string) ([]*ent.Schedule, error) {
	result, err := s.client.Schedule.Query().
		Where(schedule.Creator(creator)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get schedules by creator: %w", err)
	}
	return result, nil
}

// UpdateSchedule updates an existing schedule
func (s *ScheduleUsecase) UpdateSchedule(ctx context.Context, schedule *ent.Schedule, currentUser string) (*ent.Schedule, error) {
	// First, check if the user is allowed to edit this schedule
	existingSchedule, err := s.GetSchedule(ctx, schedule.ID)
	if err != nil {
		return nil, err
	}

	// Check if the current user is the creator or if editing by others is allowed
	if existingSchedule.Creator != currentUser && !existingSchedule.AllowEditByOthers {
		return nil, fmt.Errorf("you don't have permission to edit this schedule")
	}

	// Start building the update query
	updateQuery := s.client.Schedule.UpdateOneID(schedule.ID).
		SetName(schedule.Name).
		SetWeekDays(schedule.WeekDays).
		SetHour(schedule.Hour).
		SetMinute(schedule.Minute).
		SetOperation(schedule.Operation).
		SetEnabled(schedule.Enabled).
		SetNotifyViaServerChan(schedule.NotifyViaServerChan)

	// Only the creator can change this setting
	if existingSchedule.Creator == currentUser {
		updateQuery = updateQuery.SetAllowEditByOthers(schedule.AllowEditByOthers)
	}

	result, err := updateQuery.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update schedule: %w", err)
	}

	return result, nil
}

// DeleteSchedule deletes a schedule by ID
func (s *ScheduleUsecase) DeleteSchedule(ctx context.Context, id uuid.UUID, currentUser string) error {
	// First, check if the user is allowed to delete this schedule
	existingSchedule, err := s.GetSchedule(ctx, id)
	if err != nil {
		return err
	}

	// Only the creator can delete the schedule
	if existingSchedule.Creator != currentUser {
		return fmt.Errorf("only the creator can delete this schedule")
	}

	err = s.client.Schedule.DeleteOneID(id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
	}

	return nil
}

// GetServerChanConfig 获取Server酱配置
func (s *ScheduleUsecase) GetServerChanConfig(ctx context.Context) (*ent.ServerChanConfig, error) {
	config, err := s.client.ServerChanConfig.Query().First(ctx)
	if err != nil {
		// 如果没有配置，返回默认配置
		return &ent.ServerChanConfig{
			SendKey:     "",
			OnTemplate:  "{{.Name}} 任务执行成功，灯已开启",
			OffTemplate: "{{.Name}} 任务执行成功，灯已关闭",
			Enabled:     false,
		}, nil
	}
	return config, nil
}

// SaveServerChanConfig 保存Server酱配置
func (s *ScheduleUsecase) SaveServerChanConfig(ctx context.Context, config *ent.ServerChanConfig) error {
	existingConfig, err := s.client.ServerChanConfig.Query().First(ctx)

	if err != nil {
		// 如果没有现有配置，创建新配置
		_, err = s.client.ServerChanConfig.Create().
			SetEnabled(config.Enabled).
			SetSendKey(config.SendKey).
			SetOnTemplate(config.OnTemplate).
			SetOffTemplate(config.OffTemplate).
			Save(ctx)

		if err != nil {
			return fmt.Errorf("failed to create serverchan config: %w", err)
		}
	} else {
		// 更新现有配置
		_, err = s.client.ServerChanConfig.UpdateOneID(existingConfig.ID).
			SetEnabled(config.Enabled).
			SetSendKey(config.SendKey).
			SetOnTemplate(config.OnTemplate).
			SetOffTemplate(config.OffTemplate).
			Save(ctx)

		if err != nil {
			return fmt.Errorf("failed to update serverchan config: %w", err)
		}
	}

	return nil
}

// Close closes the database client
func (s *ScheduleUsecase) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}
