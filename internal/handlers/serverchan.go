package handlers

import (
	"cat-led/internal/ent"
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
)

// ServerChanConfig 包含Server酱的配置信息
type ServerChanConfig struct {
	Enabled      bool   `json:"enabled"`
	SendKey      string `json:"sendKey"`
	OnTemplate   string `json:"onTemplate"`
	OffTemplate  string `json:"offTemplate"`
	EmailEnabled bool   `json:"emailEnabled"`
	EmailURL     string `json:"emailURL"`
}

// GetServerChanConfig 获取Server酱配置
func GetServerChanConfig(c *gin.Context) {
	// 检查scheduleUseCase是否已初始化
	if scheduleUseCase == nil {
		c.JSON(500, gin.H{"error": "定时任务服务未初始化"})
		return
	}

	// 获取数据库客户端
	client := scheduleUseCase.GetClient()

	// 查询配置
	ctx := context.Background()
	config, err := client.ServerChanConfig.Query().First(ctx)

	// 如果没有配置，返回默认配置
	if err != nil {
		c.JSON(200, ServerChanConfig{
			Enabled:      false,
			SendKey:      "",
			OnTemplate:   "懒猫{{.Name}} 任务于{{ .Time }}执行成功，灯已开启",
			OffTemplate:  "懒猫{{.Name}} 任务于{{ .Time }}执行成功，灯已关闭",
			EmailEnabled: false,
			EmailURL:     "",
		})
		return
	}

	// 返回配置
	c.JSON(200, ServerChanConfig{
		Enabled:      config.Enabled,
		SendKey:      config.SendKey,
		OnTemplate:   config.OnTemplate,
		OffTemplate:  config.OffTemplate,
		EmailEnabled: config.EmailEnabled,
		EmailURL:     config.EmailURL,
	})
}

// SaveServerChanConfig 保存Server酱配置
func SaveServerChanConfig(c *gin.Context) {
	// 检查scheduleUseCase是否已初始化
	if scheduleUseCase == nil {
		c.JSON(500, gin.H{"error": "定时任务服务未初始化"})
		return
	}

	// 解析请求体
	var config ServerChanConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// 创建ent.ServerChanConfig对象
	entConfig := &ent.ServerChanConfig{
		Enabled:      config.Enabled,
		SendKey:      config.SendKey,
		OnTemplate:   config.OnTemplate,
		OffTemplate:  config.OffTemplate,
		EmailEnabled: config.EmailEnabled,
		EmailURL:     config.EmailURL,
	}

	// 保存配置
	ctx := context.Background()
	err := scheduleUseCase.SaveServerChanConfig(ctx, entConfig)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("保存配置失败: %v", err)})
		return
	}

	c.JSON(200, gin.H{"message": "配置已保存"})
}
