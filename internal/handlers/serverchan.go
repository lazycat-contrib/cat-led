package handlers

import (
	"context"
	"fmt"
	"net/http"

	"cat-led/internal/ent"

	"github.com/gin-gonic/gin"
)

// Default templates for ServerChan notifications (used in API responses).
const (
	defaultOnTemplateDisplay  = "懒猫{{.Name}} 任务于{{ .Time }}}执行成功，灯已开启"
	defaultOffTemplateDisplay = "懒猫{{.Name}} 任务于{{ .Time }}}执行成功，灯已关闭"
)

// ServerChanConfig represents the ServerChan configuration for API responses.
type ServerChanConfig struct {
	Enabled     bool   `json:"enabled"`
	SendKey     string `json:"sendKey"`
	OnTemplate  string `json:"onTemplate"`
	OffTemplate string `json:"offTemplate"`
}

// GetServerChanConfig returns the current ServerChan configuration.
func GetServerChanConfig(c *gin.Context) {
	if !requireScheduleUseCase(c) {
		return
	}

	ctx := context.Background()
	config, err := scheduleUseCase.GetClient().ServerChanConfig.Query().First(ctx)
	if err != nil {
		c.JSON(http.StatusOK, ServerChanConfig{
			Enabled:     false,
			SendKey:     "",
			OnTemplate:  defaultOnTemplateDisplay,
			OffTemplate: defaultOffTemplateDisplay,
		})
		return
	}

	c.JSON(http.StatusOK, ServerChanConfig{
		Enabled:     config.Enabled,
		SendKey:     config.SendKey,
		OnTemplate:  config.OnTemplate,
		OffTemplate: config.OffTemplate,
	})
}

// SaveServerChanConfig saves the ServerChan configuration.
func SaveServerChanConfig(c *gin.Context) {
	if !requireScheduleUseCase(c) {
		return
	}

	var config ServerChanConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entConfig := &ent.ServerChanConfig{
		Enabled:     config.Enabled,
		SendKey:     config.SendKey,
		OnTemplate:  config.OnTemplate,
		OffTemplate: config.OffTemplate,
	}

	ctx := context.Background()
	if err := scheduleUseCase.SaveServerChanConfig(ctx, entConfig); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("保存配置失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "配置已保存"})
}
