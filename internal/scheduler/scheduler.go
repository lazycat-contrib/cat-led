package scheduler

import (
	"context"
	"fmt"
	"log"
	"slices"
	"time"

	"cat-led/internal/biz"
	"cat-led/internal/ent"
	"cat-led/internal/ent/schedule"
	"cat-led/internal/handlers"
	"cat-led/internal/pkg/lzcnotify"
	"cat-led/internal/pkg/ntfy"
	"cat-led/internal/pkg/serverchan"
	"cat-led/internal/pkg/zlog"

	gohelper "gitee.com/linakesi/lzc-sdk/lang/go"
	users "gitee.com/linakesi/lzc-sdk/lang/go/common"
)

// Scheduler manages background task scheduling.
type Scheduler struct {
	useCase *biz.ScheduleUsecase
	logger  *zlog.Logger
}

// New creates a new Scheduler.
func New(useCase *biz.ScheduleUsecase, logger *zlog.Logger) *Scheduler {
	return &Scheduler{
		useCase: useCase,
		logger:  logger,
	}
}

// Start begins the background scheduler that checks for due tasks every minute.
func (s *Scheduler) Start() {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			s.checkSchedules()
		}
	}()
}

// checkSchedules iterates through all enabled schedules and executes any that are due.
func (s *Scheduler) checkSchedules() {
	if s.useCase == nil {
		s.logger.Warn().Msg("定时任务服务未初始化，跳过检查")
		return
	}

	now := time.Now()
	ctx := context.Background()

	allSchedules, err := s.useCase.GetAllSchedules(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("获取任务失败")
		return
	}

	currentWeekday := int(now.Weekday())
	for _, sched := range allSchedules {
		if !sched.Enabled {
			continue
		}
		if !shouldRunOnWeekday(sched.WeekDays, currentWeekday) {
			continue
		}
		if now.Hour() == sched.Hour && now.Minute() == sched.Minute {
			s.executeSchedule(ctx, sched)
		}
	}
}

// shouldRunOnWeekday checks if the schedule can run on the given weekday.
func shouldRunOnWeekday(weekDays []int, currentWeekday int) bool {
	if len(weekDays) == 0 {
		return true
	}
	return slices.Contains(weekDays, currentWeekday)
}

// executeSchedule performs the scheduled operation and sends notifications if enabled.
func (s *Scheduler) executeSchedule(ctx context.Context, sched *ent.Schedule) {
	var err error
	var status bool
	var operationName string

	switch sched.Operation {
	case schedule.OperationOn:
		status = true
		operationName = "开灯"
		err = setLedStatus(ctx, true)
	case schedule.OperationOff:
		status = false
		operationName = "关灯"
		err = setLedStatus(ctx, false)
	case schedule.OperationShutdown:
		status = false
		operationName = "关机"
		err = shutdownDevice(ctx)
	case schedule.OperationReboot:
		status = false
		operationName = "重启"
		err = rebootDevice(ctx)
	default:
		s.logger.Info().Msg("do nothing")
		return
	}

	if err != nil {
		s.logger.Error().Err(err).Str("operation", operationName).Msg("执行任务失败")
		return
	}

	s.logger.Info().Str("任务名称", sched.Name).Str("操作", operationName).Msg("执行任务成功")

	if sched.NotifyViaServerChan {
		s.sendServerChanNotification(ctx, sched.Name, status)
	}
	if sched.NotifyViaLzc {
		s.sendLzcNotification(ctx, sched.Creator, sched.Name, operationName)
	}
	if sched.NotifyViaNtfy {
		s.sendNtfyNotification(ctx, sched.Name, status)
	}

	s.disableOneTimeScheduleAfterRun(ctx, sched)
}

func (s *Scheduler) disableOneTimeScheduleAfterRun(ctx context.Context, sched *ent.Schedule) {
	if len(sched.WeekDays) > 0 {
		return
	}
	if s.useCase == nil {
		s.logger.Warn().Str("任务名称", sched.Name).Msg("定时任务服务未初始化，无法自动禁用单次任务")
		return
	}
	if err := s.useCase.SetScheduleEnabled(ctx, sched.ID, false); err != nil {
		s.logger.Error().Err(err).Str("任务名称", sched.Name).Msg("自动禁用单次任务失败")
		return
	}

	sched.Enabled = false
	s.logger.Info().Str("任务名称", sched.Name).Msg("单次任务执行后已自动禁用")
}

// sendLzcNotification sends a LazyCat built-in notification when a schedule is executed.
func (s *Scheduler) sendLzcNotification(ctx context.Context, userID, taskName, operationName string) {
	notifyCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	body := fmt.Sprintf("%s 任务执行成功，已%s", taskName, operationName)
	sent, err := lzcnotify.NotifyUser(notifyCtx, userID, "懒猫小灯任务通知", body, "")
	if err != nil {
		s.logger.Error().Err(err).Str("任务名称", taskName).Int("已发送客户端数", sent).Msg("发送懒猫内置通知失败")
		return
	}
	if sent == 0 {
		s.logger.Warn().Str("任务名称", taskName).Msg("没有可通知的在线懒猫客户端")
		return
	}
	s.logger.Info().Str("任务名称", taskName).Int("已发送客户端数", sent).Msg("发送懒猫内置通知成功")
}

// sendServerChanNotification sends a notification via ServerChan when a schedule is executed.
func (s *Scheduler) sendServerChanNotification(ctx context.Context, taskName string, status bool) {
	config, err := handlers.GetServerChanConfigForScheduler(ctx)
	if err != nil {
		s.logger.Warn().Err(err).Msg("获取Server酱配置失败，使用默认配置")
	}

	if !config.Enabled || config.SendKey == "" {
		return
	}

	pusher := serverchan.NewPusher(config.SendKey, config.OnTemplate, config.OffTemplate)
	ledContext := serverchan.LEDContext{
		Time:   time.Now(),
		Status: status,
		Name:   taskName,
	}

	if status {
		err = pusher.PushLedOpenNotify(ledContext)
	} else {
		err = pusher.PushLedCloseNotify(ledContext)
	}

	if err != nil {
		s.logger.Error().Err(err).Msg("发送Server酱通知失败")
		return
	}

	s.logger.Info().Str("任务名称", taskName).Bool("状态", status).Msg("发送Server酱通知成功")
}

// sendNtfyNotification sends a notification via ntfy when a schedule is executed.
func (s *Scheduler) sendNtfyNotification(ctx context.Context, taskName string, status bool) {
	config, err := handlers.GetNtfyConfigForScheduler(ctx)
	if err != nil {
		s.logger.Warn().Err(err).Msg("获取ntfy配置失败")
		return
	}

	if !config.Enabled || config.Topic == "" {
		return
	}

	pusher := ntfy.NewPusher(*config)
	ledContext := ntfy.LEDContext{
		Time:   time.Now(),
		Status: status,
		Name:   taskName,
	}

	if status {
		err = pusher.PushLedOpenNotify(ledContext)
	} else {
		err = pusher.PushLedCloseNotify(ledContext)
	}

	if err != nil {
		s.logger.Error().Err(err).Msg("发送ntfy通知失败")
		return
	}

	s.logger.Info().Str("任务名称", taskName).Bool("状态", status).Msg("发送ntfy通知成功")
}

// setLedStatus sets the LED power state.
func setLedStatus(ctx context.Context, status bool) error {
	gw, err := gohelper.NewAPIGateway(ctx)
	if err != nil {
		log.Printf("Error creating API gateway: %v", err)
		return err
	}
	defer gw.Close()

	_, err = gw.Box.ChangePowerLed(ctx, &users.ChangePowerLedRequest{
		PowerLed: status,
	})
	if err != nil {
		log.Printf("Error changing LED status to %v: %v", status, err)
		return err
	}

	log.Printf("LED status changed to: %v", status)
	return nil
}

// rebootDevice initiates a device reboot.
func rebootDevice(ctx context.Context) error {
	gw, err := gohelper.NewAPIGateway(ctx)
	if err != nil {
		return err
	}
	defer gw.Close()

	_, err = gw.Box.Shutdown(ctx, &users.ShutdownRequest{
		Action: users.ShutdownRequest_Reboot,
	})
	return err
}

// shutdownDevice initiates a device power off.
func shutdownDevice(ctx context.Context) error {
	gw, err := gohelper.NewAPIGateway(ctx)
	if err != nil {
		return err
	}
	defer gw.Close()

	_, err = gw.Box.Shutdown(ctx, &users.ShutdownRequest{
		Action: users.ShutdownRequest_Poweroff,
	})
	return err
}
