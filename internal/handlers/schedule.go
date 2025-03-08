package handlers

import (
	"cat-led/internal/biz"
	"cat-led/internal/ent"
	"cat-led/internal/ent/schedule"
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	gohelper "gitee.com/linakesi/lzc-sdk/lang/go"
	users "gitee.com/linakesi/lzc-sdk/lang/go/common"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var (
	// 全局的scheduleUseCase实例
	scheduleUseCase *biz.ScheduleUsecase
	schOnce         sync.Once
	// 互斥锁用于保护scheduleUseCase的访问
	scheduleMutex sync.Mutex
)

// InitScheduleUseCase 初始化scheduleUseCase
func InitScheduleUseCase(dbPath string) {
	schOnce.Do(func() {
		scheduleUseCase = biz.NewScheduleUseCase(dbPath)
		if scheduleUseCase == nil {
			log.Println("初始化scheduleUseCase失败")
		} else {
			log.Println("成功初始化scheduleUseCase")
		}
	})
}

// 将前端Schedule转换为ent.Schedule
func convertToEntSchedule(frontendSchedule map[string]interface{}, creatorID string) (*ent.Schedule, error) {
	name, _ := frontendSchedule["name"].(string)
	enabled, _ := frontendSchedule["enabled"].(bool)
	allowEdit, _ := frontendSchedule["allowEdit"].(bool)

	// 处理重复日期
	var weekDays []int
	if repeatDaysInterface, ok := frontendSchedule["repeatDays"].([]interface{}); ok {
		for _, day := range repeatDaysInterface {
			if dayInt, ok := day.(float64); ok {
				weekDays = append(weekDays, int(dayInt))
			}
		}
	}

	// 处理时间 - 直接使用小时和分钟
	hour, minute := 0, 0
	if hourFloat, ok := frontendSchedule["hour"].(float64); ok {
		hour = int(hourFloat)
	}
	if minuteFloat, ok := frontendSchedule["minute"].(float64); ok {
		minute = int(minuteFloat)
	}

	// 确定操作类型 (on/off)
	operation := schedule.OperationOn
	if opStr, ok := frontendSchedule["operation"].(string); ok {
		if opStr == "off" {
			operation = schedule.OperationOff
		} else if opStr == "on" {
			operation = schedule.OperationOn
		}
	}

	// 创建Schedule实体
	s := &ent.Schedule{
		Name:              name,
		Creator:           creatorID,
		WeekDays:          weekDays,
		Hour:              hour,
		Minute:            minute,
		Operation:         operation,
		Enabled:           enabled,
		AllowEditByOthers: allowEdit,
	}

	return s, nil
}

// 将ent.Schedule转换为前端Schedule格式
func convertToFrontendSchedule(entSchedule *ent.Schedule) map[string]interface{} {
	// 前端Schedule格式
	return map[string]interface{}{
		"id":           entSchedule.ID.String(),
		"name":         entSchedule.Name,
		"hour":         entSchedule.Hour,
		"minute":       entSchedule.Minute,
		"enabled":      entSchedule.Enabled,
		"repeatDays":   entSchedule.WeekDays,
		"creatorId":    entSchedule.Creator,
		"allowEdit":    entSchedule.AllowEditByOthers,
		"operation":    string(entSchedule.Operation),
		"createdAt":    time.Now().Format(time.RFC3339),
		"lastModified": time.Now().Format(time.RFC3339),
	}
}

// GetSchedules 获取所有定时任务
func GetSchedules(c *gin.Context) {
	if scheduleUseCase == nil {
		c.JSON(500, gin.H{"error": "定时任务服务未初始化"})
		return
	}

	userID := c.GetHeader("x-hc-user-id")
	if userID == "" {
		// 使用正确的方式获取用户信息
		gw, err := gohelper.NewAPIGateway(c.Request.Context())
		if err != nil {
			c.JSON(401, gin.H{"error": "未授权"})
			return
		}
		defer gw.Close()

		userInfo, err := gw.Users.QueryUserInfo(c.Request.Context(), &users.UserID{Uid: userID})
		if err == nil && userInfo != nil && userInfo.Uid != "" {
			userID = userInfo.Uid
		}
	}

	if userID == "" {
		c.JSON(401, gin.H{"error": "未授权"})
		return
	}

	// 使用context进行数据库操作
	ctx := context.Background()

	// 获取用户创建的任务
	userSchedules, err := scheduleUseCase.GetSchedulesByCreator(ctx, userID)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("获取任务失败: %v", err)})
		return
	}

	// 获取所有允许编辑的任务
	allSchedules, err := scheduleUseCase.GetAllSchedules(ctx)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("获取任务失败: %v", err)})
		return
	}

	// 组合结果
	result := make([]map[string]interface{}, 0)

	// 添加用户的任务
	for _, s := range userSchedules {
		result = append(result, convertToFrontendSchedule(s))
	}

	// 添加允许编辑的其他任务
	for _, s := range allSchedules {
		// 跳过已添加的用户任务
		if s.Creator == userID {
			continue
		}

		// 只添加允许编辑的任务
		if s.AllowEditByOthers {
			result = append(result, convertToFrontendSchedule(s))
		}
	}

	c.JSON(200, result)
}

// CreateSchedule 创建LED定时任务
func CreateSchedule(c *gin.Context) {
	if scheduleUseCase == nil {
		c.JSON(500, gin.H{"error": "定时任务服务未初始化"})
		return
	}

	// 获取用户ID
	userID := c.GetHeader("x-hc-user-id")
	if userID == "" {
		// 使用正确的方式获取用户信息
		gw, err := gohelper.NewAPIGateway(c.Request.Context())
		if err != nil {
			c.JSON(401, gin.H{"error": "未授权"})
			return
		}
		defer gw.Close()

		userInfo, err := gw.Users.QueryUserInfo(c.Request.Context(), &users.UserID{Uid: userID})
		if err == nil && userInfo != nil && userInfo.Uid != "" {
			userID = userInfo.Uid
		}
	}

	if userID == "" {
		c.JSON(401, gin.H{"error": "未授权"})
		return
	}

	// 解析请求体
	var frontendSchedule map[string]interface{}
	if err := c.ShouldBindJSON(&frontendSchedule); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// 转换为ent.Schedule
	entSchedule, err := convertToEntSchedule(frontendSchedule, userID)
	if err != nil {
		c.JSON(400, gin.H{"error": fmt.Sprintf("解析任务数据失败: %v", err)})
		return
	}

	// 创建任务
	ctx := context.Background()
	createdSchedule, err := scheduleUseCase.CreateSchedule(ctx, entSchedule)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("创建任务失败: %v", err)})
		return
	}

	// 返回创建的任务
	c.JSON(201, convertToFrontendSchedule(createdSchedule))
}

// UpdateSchedule 更新LED定时任务
func UpdateSchedule(c *gin.Context) {
	if scheduleUseCase == nil {
		c.JSON(500, gin.H{"error": "定时任务服务未初始化"})
		return
	}

	// 获取用户ID
	userID := c.GetHeader("x-hc-user-id")
	if userID == "" {
		// 使用正确的方式获取用户信息
		gw, err := gohelper.NewAPIGateway(c.Request.Context())
		if err != nil {
			c.JSON(401, gin.H{"error": "未授权"})
			return
		}
		defer gw.Close()

		userInfo, err := gw.Users.QueryUserInfo(c.Request.Context(), &users.UserID{Uid: userID})
		if err == nil && userInfo != nil && userInfo.Uid != "" {
			userID = userInfo.Uid
		}
	}

	if userID == "" {
		c.JSON(401, gin.H{"error": "未授权"})
		return
	}

	// 获取任务ID
	scheduleID := c.Param("id")
	scheduleUUID, err := uuid.Parse(scheduleID)
	if err != nil {
		c.JSON(400, gin.H{"error": "无效的任务ID"})
		return
	}

	// 解析请求体
	var frontendSchedule map[string]interface{}
	if err := c.ShouldBindJSON(&frontendSchedule); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// 转换为ent.Schedule
	entSchedule, err := convertToEntSchedule(frontendSchedule, userID)
	if err != nil {
		c.JSON(400, gin.H{"error": fmt.Sprintf("解析任务数据失败: %v", err)})
		return
	}

	// 设置ID
	entSchedule.ID = scheduleUUID

	// 更新任务
	ctx := context.Background()
	updatedSchedule, err := scheduleUseCase.UpdateSchedule(ctx, entSchedule, userID)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("更新任务失败: %v", err)})
		return
	}

	// 返回更新后的任务
	c.JSON(200, convertToFrontendSchedule(updatedSchedule))
}

// DeleteSchedule 删除LED定时任务
func DeleteSchedule(c *gin.Context) {
	if scheduleUseCase == nil {
		c.JSON(500, gin.H{"error": "定时任务服务未初始化"})
		return
	}

	// 获取用户ID
	userID := c.GetHeader("x-hc-user-id")
	if userID == "" {
		// 使用正确的方式获取用户信息
		gw, err := gohelper.NewAPIGateway(c.Request.Context())
		if err != nil {
			c.JSON(401, gin.H{"error": "未授权"})
			return
		}
		defer gw.Close()

		userInfo, err := gw.Users.QueryUserInfo(c.Request.Context(), &users.UserID{Uid: userID})
		if err == nil && userInfo != nil && userInfo.Uid != "" {
			userID = userInfo.Uid
		}
	}

	if userID == "" {
		c.JSON(401, gin.H{"error": "未授权"})
		return
	}

	// 获取任务ID
	scheduleID := c.Param("id")
	scheduleUUID, err := uuid.Parse(scheduleID)
	if err != nil {
		c.JSON(400, gin.H{"error": "无效的任务ID"})
		return
	}

	// 删除任务
	ctx := context.Background()
	err = scheduleUseCase.DeleteSchedule(ctx, scheduleUUID, userID)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("删除任务失败: %v", err)})
		return
	}

	c.JSON(200, gin.H{"message": "任务已删除"})
}

// SetLedStatus 设置LED状态
func SetLedStatus(ctx context.Context, status bool) error {
	// 使用正确的API来设置LED状态
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

// InitScheduler 初始化定时任务调度器
func InitScheduler() {
	// 检查任务的定时器
	go func() {
		// 每分钟检查一次任务
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				checkSchedules()
			}
		}
	}()
}

// 检查是否有需要执行的任务
func checkSchedules() {
	if scheduleUseCase == nil {
		log.Println("定时任务服务未初始化，跳过检查")
		return
	}

	now := time.Now()
	ctx := context.Background()

	// 获取所有任务
	allSchedules, err := scheduleUseCase.GetAllSchedules(ctx)
	if err != nil {
		log.Printf("获取任务失败: %v", err)
		return
	}

	for _, s := range allSchedules {
		// 跳过禁用的任务
		if !s.Enabled {
			continue
		}

		// 检查是否是当前星期几
		weekday := int(now.Weekday())
		shouldRun := false
		for _, d := range s.WeekDays {
			if d == weekday {
				shouldRun = true
				break
			}
		}

		if !shouldRun {
			continue
		}

		// 检查是否是设定的时间
		if now.Hour() == s.Hour && now.Minute() == s.Minute {
			// 执行任务
			status := s.Operation == schedule.OperationOn
			if err := SetLedStatus(ctx, status); err != nil {
				log.Printf("执行任务失败: %v", err)
			} else {
				log.Printf("执行任务成功: %s, 状态: %v", s.Name, status)
			}
		}
	}
}
