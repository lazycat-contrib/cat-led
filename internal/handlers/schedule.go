package handlers

import (
	"context"
	"log"
	"sync"
	"time"

	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	gohelper "gitee.com/linakesi/lzc-sdk/lang/go"
	users "gitee.com/linakesi/lzc-sdk/lang/go/common"
	"github.com/gin-gonic/gin"
)

// 存储文件路径
const (
	dataDir       = "/lzcapp/run/data"
	schedulesFile = "led_schedules.json"
)

// LedSchedule 表示LED定时任务
type LedSchedule struct {
	ID           string    `json:"id"`           // 任务ID
	Name         string    `json:"name"`         // 任务名称
	StartTime    time.Time `json:"startTime"`    // 开始时间
	EndTime      time.Time `json:"endTime"`      // 结束时间
	Enabled      bool      `json:"enabled"`      // 是否启用
	RepeatDays   []int     `json:"repeatDays"`   // 重复的星期几 (0-6, 0是周日)
	CreatorID    string    `json:"creatorId"`    // 创建者ID
	AllowEdit    bool      `json:"allowEdit"`    // 是否允许他人编辑
	CreatedAt    time.Time `json:"createdAt"`    // 创建时间
	LastModified time.Time `json:"lastModified"` // 最后修改时间
}

var (
	// schedules 存储所有LED定时任务
	schedules      = make(map[string]LedSchedule)
	schedulesMutex sync.Mutex
)

// 确保数据目录存在
func ensureDataDir() error {
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		// 尝试创建目录
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			log.Printf("无法创建数据目录 %s: %v", dataDir, err)
			// 如果无法创建标准目录，尝试使用当前目录
			if err := os.MkdirAll("./data", 0755); err != nil {
				return err
			}
			return nil
		}
	}
	return nil
}

// 获取存储文件的完整路径
func getStorageFilePath() string {
	// 检查标准目录是否可写
	if _, err := os.Stat(dataDir); err == nil {
		return filepath.Join(dataDir, schedulesFile)
	}
	// 否则使用当前目录
	return filepath.Join("./data", schedulesFile)
}

// LoadSchedules 从文件加载定时任务
func LoadSchedules() error {
	// 确保数据目录存在
	if err := ensureDataDir(); err != nil {
		log.Printf("确保数据目录存在时出错: %v", err)
		return err
	}

	filePath := getStorageFilePath()

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("定时任务文件不存在，将创建新文件")
		return nil
	}

	// 读取文件
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Printf("读取定时任务文件错误: %v", err)
		return err
	}

	// 如果文件为空，返回
	if len(data) == 0 {
		log.Printf("定时任务文件为空")
		return nil
	}

	// 解析JSON
	var loadedSchedules map[string]LedSchedule
	if err := json.Unmarshal(data, &loadedSchedules); err != nil {
		log.Printf("解析定时任务JSON错误: %v", err)
		return err
	}

	// 更新内存中的任务
	schedulesMutex.Lock()
	schedules = loadedSchedules
	schedulesMutex.Unlock()

	log.Printf("成功加载了 %d 个定时任务", len(loadedSchedules))
	return nil
}

// SaveSchedules 将定时任务保存到文件
func SaveSchedules() error {
	// 确保数据目录存在
	if err := ensureDataDir(); err != nil {
		log.Printf("确保数据目录存在时出错: %v", err)
		return err
	}

	filePath := getStorageFilePath()

	schedulesMutex.Lock()
	data, err := json.MarshalIndent(schedules, "", "  ")
	schedulesMutex.Unlock()

	if err != nil {
		log.Printf("序列化定时任务错误: %v", err)
		return err
	}

	if err := ioutil.WriteFile(filePath, data, 0644); err != nil {
		log.Printf("写入定时任务文件错误: %v", err)
		return err
	}

	log.Printf("成功保存了 %d 个定时任务", len(schedules))
	return nil
}

// GetSchedules 获取所有LED定时任务
func GetSchedules(c *gin.Context) {
	userID := c.GetHeader("x-hc-user-id")

	schedulesMutex.Lock()
	defer schedulesMutex.Unlock()

	var result []LedSchedule
	for _, schedule := range schedules {
		// 如果是创建者或者允许编辑，则返回该任务
		if schedule.CreatorID == userID || schedule.AllowEdit {
			result = append(result, schedule)
		}
	}

	c.JSON(200, result)
}

// CreateSchedule 创建LED定时任务
func CreateSchedule(c *gin.Context) {
	var schedule LedSchedule
	if err := c.ShouldBindJSON(&schedule); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetHeader("x-hc-user-id")

	// 设置任务信息
	schedule.ID = time.Now().Format("20060102150405") // 使用时间戳作为ID
	schedule.CreatorID = userID
	schedule.CreatedAt = time.Now()
	schedule.LastModified = time.Now()

	schedulesMutex.Lock()
	schedules[schedule.ID] = schedule
	schedulesMutex.Unlock()

	// 保存到文件
	if err := SaveSchedules(); err != nil {
		log.Printf("保存定时任务失败: %v", err)
		// 继续执行，不返回错误
	}

	c.JSON(201, schedule)
}

// UpdateSchedule 更新LED定时任务
func UpdateSchedule(c *gin.Context) {
	scheduleID := c.Param("id")

	schedulesMutex.Lock()
	schedule, exists := schedules[scheduleID]
	if !exists {
		schedulesMutex.Unlock()
		c.JSON(404, gin.H{"error": "任务不存在"})
		return
	}

	userID := c.GetHeader("x-hc-user-id")

	// 检查是否有权限编辑
	if schedule.CreatorID != userID && !schedule.AllowEdit {
		schedulesMutex.Unlock()
		c.JSON(403, gin.H{"error": "无权限编辑此任务"})
		return
	}

	var updatedSchedule LedSchedule
	if err := c.ShouldBindJSON(&updatedSchedule); err != nil {
		schedulesMutex.Unlock()
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// 更新任务信息，保留原创建者和创建时间
	updatedSchedule.ID = scheduleID
	updatedSchedule.CreatorID = schedule.CreatorID
	updatedSchedule.CreatedAt = schedule.CreatedAt
	updatedSchedule.LastModified = time.Now()

	schedules[scheduleID] = updatedSchedule
	schedulesMutex.Unlock()

	// 保存到文件
	if err := SaveSchedules(); err != nil {
		log.Printf("保存定时任务失败: %v", err)
		// 继续执行，不返回错误
	}

	c.JSON(200, updatedSchedule)
}

// DeleteSchedule 删除LED定时任务
func DeleteSchedule(c *gin.Context) {
	scheduleID := c.Param("id")

	schedulesMutex.Lock()
	schedule, exists := schedules[scheduleID]
	if !exists {
		schedulesMutex.Unlock()
		c.JSON(404, gin.H{"error": "任务不存在"})
		return
	}

	userID := c.GetHeader("x-hc-user-id")

	// 检查是否有权限删除（只有创建者可以删除）
	if schedule.CreatorID != userID {
		schedulesMutex.Unlock()
		c.JSON(403, gin.H{"error": "无权限删除此任务"})
		return
	}

	delete(schedules, scheduleID)
	schedulesMutex.Unlock()

	// 保存到文件
	if err := SaveSchedules(); err != nil {
		log.Printf("保存定时任务失败: %v", err)
		// 继续执行，不返回错误
	}

	c.JSON(200, gin.H{"message": "任务已删除"})
}

// SetLedStatus 设置LED状态
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

	// 更新全局LED状态变量
	ledMutex.Lock()
	ledStatus = status
	ledMutex.Unlock()

	log.Printf("LED status changed to: %v", status)
	return nil
}

// InitScheduler 初始化定时任务调度器
func InitScheduler() {
	// 加载已保存的定时任务
	if err := LoadSchedules(); err != nil {
		log.Printf("加载定时任务失败: %v", err)
	}

	go func() {
		for {
			checkSchedules()
			time.Sleep(1 * time.Minute) // 每分钟检查一次
		}
	}()
}

// checkSchedules 检查并执行定时任务
func checkSchedules() {
	now := time.Now()
	weekday := int(now.Weekday())

	schedulesMutex.Lock()
	defer schedulesMutex.Unlock()

	for _, schedule := range schedules {
		if !schedule.Enabled {
			continue
		}

		// 检查今天是否在重复日期内
		dayMatched := false
		for _, day := range schedule.RepeatDays {
			if day == weekday {
				dayMatched = true
				break
			}
		}

		if !dayMatched && len(schedule.RepeatDays) > 0 {
			continue
		}

		// 只比较时分，忽略日期部分
		currentTime := time.Date(0, 0, 0, now.Hour(), now.Minute(), 0, 0, time.Local)
		startTime := time.Date(0, 0, 0, schedule.StartTime.Hour(), schedule.StartTime.Minute(), 0, 0, time.Local)
		endTime := time.Date(0, 0, 0, schedule.EndTime.Hour(), schedule.EndTime.Minute(), 0, 0, time.Local)

		// 如果当前时间等于开始时间，打开LED
		if currentTime.Equal(startTime) {
			SetLedStatus(context.Background(), true)
		}

		// 如果当前时间等于结束时间，关闭LED
		if currentTime.Equal(endTime) {
			SetLedStatus(context.Background(), false)
		}
	}
}
