package handlers

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"cat-led/internal/auth"
	"cat-led/internal/biz"
	"cat-led/internal/ent"
	"cat-led/internal/ent/schedule"
	"cat-led/internal/pkg/zlog"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var (
	scheduleUseCase *biz.ScheduleUsecase
	schOnce         sync.Once
)

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

func GetScheduleUseCase() *biz.ScheduleUsecase {
	return scheduleUseCase
}

func getUserID(c *gin.Context) string {
	return auth.UserID(c)
}

func requireUserID(c *gin.Context) (string, bool) {
	userID := getUserID(c)
	if userID == "" {
		c.JSON(401, gin.H{"error": "未授权"})
		return "", false
	}
	return userID, true
}

func requireScheduleUseCase(c *gin.Context) bool {
	if scheduleUseCase == nil {
		c.JSON(500, gin.H{"error": "定时任务服务未初始化"})
		return false
	}
	return true
}

func convertToEntSchedule(frontendSchedule map[string]interface{}, creatorID string) (*ent.Schedule, error) {
	name, _ := frontendSchedule["name"].(string)
	enabled, _ := frontendSchedule["enabled"].(bool)
	allowEdit, _ := frontendSchedule["allowEdit"].(bool)
	notifyViaServerChan, _ := frontendSchedule["notifyViaServerChan"].(bool)
	notifyViaLzc, _ := frontendSchedule["notifyViaLzc"].(bool)
	notifyViaNtfy, _ := frontendSchedule["notifyViaNtfy"].(bool)

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
		NotifyViaLzc:        notifyViaLzc,
		NotifyViaNtfy:       notifyViaNtfy,
	}, nil
}

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

func extractTime(frontendSchedule map[string]interface{}) (hour, minute int) {
	if hourFloat, ok := frontendSchedule["hour"].(float64); ok {
		hour = int(hourFloat)
	}
	if minuteFloat, ok := frontendSchedule["minute"].(float64); ok {
		minute = int(minuteFloat)
	}
	return hour, minute
}

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
		"notifyViaLzc":        entSchedule.NotifyViaLzc,
		"notifyViaNtfy":       entSchedule.NotifyViaNtfy,
		"createdAt":           now,
		"lastModified":        now,
	}
}

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

	if op, _ := frontendSchedule["operation"].(string); op == "shutdown" || op == "reboot" {
		if !requirePowerAdmin(c) {
			return
		}
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

	existing, err := scheduleUseCase.GetSchedule(c.Request.Context(), scheduleUUID)
	if err != nil {
		c.JSON(404, gin.H{"error": "任务不存在"})
		return
	}
	if (existing.Operation == "shutdown" || existing.Operation == "reboot") && !requirePowerAdmin(c) {
		return
	}
	var frontendSchedule map[string]interface{}
	if err := c.ShouldBindJSON(&frontendSchedule); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if op, _ := frontendSchedule["operation"].(string); op == "shutdown" || op == "reboot" {
		if !requirePowerAdmin(c) {
			return
		}
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

	existing, err := scheduleUseCase.GetSchedule(c.Request.Context(), scheduleUUID)
	if err != nil {
		c.JSON(404, gin.H{"error": "任务不存在"})
		return
	}
	if (existing.Operation == "shutdown" || existing.Operation == "reboot") && !requirePowerAdmin(c) {
		return
	}
	ctx := context.Background()
	if err := scheduleUseCase.DeleteSchedule(ctx, scheduleUUID, userID); err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("删除任务失败: %v", err)})
		return
	}

	c.JSON(200, gin.H{"message": "任务已删除"})
}

func requirePowerAdmin(c *gin.Context) bool {
	allowed, err := auth.IsLazyCatAdmin(c.Request.Context(), auth.UserID(c))
	if err != nil {
		c.JSON(503, gin.H{"error": "暂时无法核实懒猫管理员身份"})
		return false
	}
	if !allowed {
		c.JSON(403, gin.H{"error": "仅懒猫管理员可设置电源操作"})
		return false
	}
	return true
}
