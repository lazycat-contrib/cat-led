package handlers

import (
	"context"
	"fmt"
	"log"
	"slices"
	"sync"
	"time"

	"cat-led/internal/biz"
	"cat-led/internal/ent"
	"cat-led/internal/ent/schedule"
	"cat-led/internal/pkg/serverchan"
	"cat-led/internal/pkg/zlog"

	gohelper "gitee.com/linakesi/lzc-sdk/lang/go"
	users "gitee.com/linakesi/lzc-sdk/lang/go/common"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var (
	scheduleUseCase *biz.ScheduleUsecase
	schOnce         sync.Once
)

// InitScheduleUseCase initializes the global scheduleUseCase instance.
func InitScheduleUseCase(dbPath string, logger *zlog.Logger) {
	schOnce.Do(func() {
		scheduleUseCase = biz.NewScheduleUseCase(dbPath, logger)
		if scheduleUseCase == nil {
			log.Println("初始化scheduleUseCase失败")
		} else {
			log.Println("成功初始化scheduleUseCase")
		}
	})
}

// getUserID extracts the user ID from the request context.
// It first checks the x-hc-user-id header, then falls back to querying the API gateway.
// Returns an empty string if the user cannot be identified.
func getUserID(c *gin.Context) string {
	userID := c.GetHeader("x-hc-user-id")
	if userID != "" {
		return userID
	}

	gw, err := gohelper.NewAPIGateway(c.Request.Context())
	if err != nil {
		return ""
	}
	defer gw.Close()

	userInfo, err := gw.Users.QueryUserInfo(c.Request.Context(), &users.UserID{Uid: userID})
	if err == nil && userInfo != nil && userInfo.Uid != "" {
		return userInfo.Uid
	}
	return ""
}

// requireUserID extracts the user ID and sends an unauthorized response if not found.
// Returns the user ID and true if successful, or empty string and false if unauthorized.
func requireUserID(c *gin.Context) (string, bool) {
	userID := getUserID(c)
	if userID == "" {
		c.JSON(401, gin.H{"error": "未授权"})
		return "", false
	}
	return userID, true
}

// requireScheduleUseCase checks if scheduleUseCase is initialized and sends an error response if not.
// Returns true if initialized, false otherwise.
func requireScheduleUseCase(c *gin.Context) bool {
	if scheduleUseCase == nil {
		c.JSON(500, gin.H{"error": "定时任务服务未初始化"})
		return false
	}
	return true
}

// convertToEntSchedule transforms a frontend schedule map into an ent.Schedule entity.
func convertToEntSchedule(frontendSchedule map[string]interface{}, creatorID string) (*ent.Schedule, error) {
	name, _ := frontendSchedule["name"].(string)
	enabled, _ := frontendSchedule["enabled"].(bool)
	allowEdit, _ := frontendSchedule["allowEdit"].(bool)
	notifyViaServerChan, _ := frontendSchedule["notifyViaServerChan"].(bool)

	weekDays := extractWeekDays(frontendSchedule)
	hour, minute := extractTime(frontendSchedule)
	operation := parseOperation(frontendSchedule)

	return &ent.Schedule{
		Name:                name,
		Creator:             creatorID,
		WeekDays:            weekDays,
		Hour:                hour,
		Minute:              minute,
		Operation:           operation,
		Enabled:             enabled,
		AllowEditByOthers:   allowEdit,
		NotifyViaServerChan: notifyViaServerChan,
	}, nil
}

// extractWeekDays parses the repeatDays field from the frontend schedule.
func extractWeekDays(frontendSchedule map[string]interface{}) []int {
	var weekDays []int
	repeatDaysInterface, ok := frontendSchedule["repeatDays"].([]interface{})
	if !ok {
		return weekDays
	}
	for _, day := range repeatDaysInterface {
		if dayInt, ok := day.(float64); ok {
			weekDays = append(weekDays, int(dayInt))
		}
	}
	return weekDays
}

// extractTime parses hour and minute from the frontend schedule.
func extractTime(frontendSchedule map[string]interface{}) (hour, minute int) {
	if hourFloat, ok := frontendSchedule["hour"].(float64); ok {
		hour = int(hourFloat)
	}
	if minuteFloat, ok := frontendSchedule["minute"].(float64); ok {
		minute = int(minuteFloat)
	}
	return hour, minute
}

// parseOperation converts the operation string to schedule.Operation enum.
func parseOperation(frontendSchedule map[string]interface{}) schedule.Operation {
	opStr, ok := frontendSchedule["operation"].(string)
	if !ok {
		return schedule.OperationOn
	}

	switch opStr {
	case "off":
		return schedule.OperationOff
	case "shutdown":
		return schedule.OperationShutdown
	case "reboot":
		return schedule.OperationReboot
	default:
		return schedule.OperationOn
	}
}

// convertToFrontendSchedule transforms an ent.Schedule entity into the frontend format.
func convertToFrontendSchedule(entSchedule *ent.Schedule) map[string]interface{} {
	now := time.Now().Format(time.RFC3339)
	return map[string]interface{}{
		"id":                  entSchedule.ID.String(),
		"name":                entSchedule.Name,
		"hour":                entSchedule.Hour,
		"minute":              entSchedule.Minute,
		"enabled":             entSchedule.Enabled,
		"repeatDays":          entSchedule.WeekDays,
		"creatorId":           entSchedule.Creator,
		"allowEdit":           entSchedule.AllowEditByOthers,
		"operation":           string(entSchedule.Operation),
		"notifyViaServerChan": entSchedule.NotifyViaServerChan,
		"createdAt":           now,
		"lastModified":        now,
	}
}

// GetSchedules returns all schedules visible to the current user.
// This includes the user's own schedules and other schedules marked as editable by others.
func GetSchedules(c *gin.Context) {
	if !requireScheduleUseCase(c) {
		return
	}

	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	ctx := context.Background()

	userSchedules, err := scheduleUseCase.GetSchedulesByCreator(ctx, userID)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("获取任务失败: %v", err)})
		return
	}

	allSchedules, err := scheduleUseCase.GetAllSchedules(ctx)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("获取任务失败: %v", err)})
		return
	}

	result := buildScheduleList(userSchedules, allSchedules, userID)
	c.JSON(200, result)
}

// buildScheduleList combines user schedules with other editable schedules.
func buildScheduleList(userSchedules, allSchedules []*ent.Schedule, userID string) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(userSchedules))

	for _, s := range userSchedules {
		result = append(result, convertToFrontendSchedule(s))
	}

	for _, s := range allSchedules {
		if s.Creator == userID {
			continue
		}
		if s.AllowEditByOthers {
			result = append(result, convertToFrontendSchedule(s))
		}
	}

	return result
}

// CreateSchedule creates a new LED schedule.
func CreateSchedule(c *gin.Context) {
	if !requireScheduleUseCase(c) {
		return
	}

	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	var frontendSchedule map[string]interface{}
	if err := c.ShouldBindJSON(&frontendSchedule); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	entSchedule, err := convertToEntSchedule(frontendSchedule, userID)
	if err != nil {
		c.JSON(400, gin.H{"error": fmt.Sprintf("解析任务数据失败: %v", err)})
		return
	}

	ctx := context.Background()
	createdSchedule, err := scheduleUseCase.CreateSchedule(ctx, entSchedule)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("创建任务失败: %v", err)})
		return
	}

	c.JSON(201, convertToFrontendSchedule(createdSchedule))
}

// UpdateSchedule updates an existing LED schedule.
func UpdateSchedule(c *gin.Context) {
	if !requireScheduleUseCase(c) {
		return
	}

	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	scheduleUUID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "无效的任务ID"})
		return
	}

	var frontendSchedule map[string]interface{}
	if err := c.ShouldBindJSON(&frontendSchedule); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	entSchedule, err := convertToEntSchedule(frontendSchedule, userID)
	if err != nil {
		c.JSON(400, gin.H{"error": fmt.Sprintf("解析任务数据失败: %v", err)})
		return
	}
	entSchedule.ID = scheduleUUID

	ctx := context.Background()
	updatedSchedule, err := scheduleUseCase.UpdateSchedule(ctx, entSchedule, userID)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("更新任务失败: %v", err)})
		return
	}

	c.JSON(200, convertToFrontendSchedule(updatedSchedule))
}

// DeleteSchedule deletes an LED schedule by ID.
func DeleteSchedule(c *gin.Context) {
	if !requireScheduleUseCase(c) {
		return
	}

	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	scheduleUUID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "无效的任务ID"})
		return
	}

	ctx := context.Background()
	if err := scheduleUseCase.DeleteSchedule(ctx, scheduleUUID, userID); err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("删除任务失败: %v", err)})
		return
	}

	c.JSON(200, gin.H{"message": "任务已删除"})
}

// SetLedStatus sets the LED power state.
func SetLedStatus(ctx context.Context, status bool) error {
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

// Reboot initiates a device reboot.
func Reboot(ctx context.Context) error {
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

// Shutdown initiates a device power off.
func Shutdown(ctx context.Context) error {
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

// InitScheduler starts the background scheduler that checks for due tasks every minute.
func InitScheduler(logger *zlog.Logger) {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			checkSchedules(logger)
		}
	}()
}

// checkSchedules iterates through all enabled schedules and executes any that are due.
func checkSchedules(logger *zlog.Logger) {
	if scheduleUseCase == nil {
		logger.Warn().Msg("定时任务服务未初始化，跳过检查")
		return
	}

	now := time.Now()
	ctx := context.Background()

	allSchedules, err := scheduleUseCase.GetAllSchedules(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("获取任务失败")
		return
	}

	currentWeekday := int(now.Weekday())
	for _, s := range allSchedules {
		if !s.Enabled {
			continue
		}
		if !shouldRunOnWeekday(s.WeekDays, currentWeekday) {
			continue
		}
		if now.Hour() == s.Hour && now.Minute() == s.Minute {
			executeSchedule(ctx, logger, s)
		}
	}
}

// shouldRunOnWeekday checks if the given weekday is in the schedule's weekdays list.
func shouldRunOnWeekday(weekDays []int, currentWeekday int) bool {
	return slices.Contains(weekDays, currentWeekday)
}

// executeSchedule performs the scheduled operation and sends notifications if enabled.
func executeSchedule(ctx context.Context, logger *zlog.Logger, s *ent.Schedule) {
	var err error
	var status bool
	var operationName string

	switch s.Operation {
	case schedule.OperationOn:
		status = true
		operationName = "开灯"
		err = SetLedStatus(ctx, true)
	case schedule.OperationOff:
		status = false
		operationName = "关灯"
		err = SetLedStatus(ctx, false)
	case schedule.OperationShutdown:
		status = false
		operationName = "关机"
		err = Shutdown(ctx)
	case schedule.OperationReboot:
		status = false
		operationName = "重启"
		err = Reboot(ctx)
	default:
		logger.Info().Msg("do nothing")
		return
	}

	if err != nil {
		logger.Error().Err(err).Str("operation", operationName).Msg("执行任务失败")
		return
	}

	logger.Info().Str("任务名称", s.Name).Str("操作", operationName).Msg("执行任务成功")

	if s.NotifyViaServerChan {
		sendServerChanNotification(ctx, logger, s.Name, status)
	}
}

// sendServerChanNotification sends a notification via ServerChan when a schedule is executed.
func sendServerChanNotification(ctx context.Context, logger *zlog.Logger, taskName string, status bool) {
	if scheduleUseCase == nil {
		logger.Error().Msg("定时任务服务未初始化")
		return
	}

	config, err := getServerChanConfig(ctx)
	if err != nil {
		logger.Warn().Err(err).Msg("获取Server酱配置失败，使用默认配置")
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
		logger.Error().Err(err).Msg("发送Server酱通知失败")
		return
	}

	logger.Info().Str("任务名称", taskName).Bool("状态", status).Msg("发送Server酱通知成功")
}

// getServerChanConfig retrieves the ServerChan configuration from the database.
func getServerChanConfig(ctx context.Context) (*ent.ServerChanConfig, error) {
	client := scheduleUseCase.GetClient()
	config, err := client.ServerChanConfig.Query().First(ctx)
	if err != nil {
		return &ent.ServerChanConfig{
			SendKey:     "",
			OnTemplate:  "{{.Name}} 任务执行成功，灯已开启",
			OffTemplate: "{{.Name}} 任务执行成功，灯已关闭",
			Enabled:     false,
		}, err
	}
	return config, nil
}
