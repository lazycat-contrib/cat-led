package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"cat-led/internal/ent"
	"cat-led/internal/pkg/ntfy"

	"github.com/gin-gonic/gin"
)

func GetNtfyConfig(c *gin.Context) {
	if !requireScheduleUseCase(c) {
		return
	}

	ctx := context.Background()
	client := scheduleUseCase.GetClient()
	config, err := client.NtfyConfig.Query().First(ctx)
	if err != nil {
		c.JSON(http.StatusOK, ntfy.Config{
			Enabled:     false,
			ServerURL:   "https://ntfy.sh",
			OnTemplate:  "{{.Name}} 任务执行成功，灯已开启",
			OffTemplate: "{{.Name}} 任务执行成功，灯已关闭",
		})
		return
	}

	c.JSON(http.StatusOK, ntfy.Config{
		Enabled:     config.Enabled,
		ServerURL:   config.ServerURL,
		Topic:       config.Topic,
		Token:       config.Token,
		OnTemplate:  config.OnTemplate,
		OffTemplate: config.OffTemplate,
	})
}

func SaveNtfyConfig(c *gin.Context) {
	if !requireScheduleUseCase(c) {
		return
	}

	var cfg ntfy.Config
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()
	client := scheduleUseCase.GetClient()
	existing, err := client.NtfyConfig.Query().First(ctx)

	if err != nil {
		_, err = client.NtfyConfig.Create().
			SetEnabled(cfg.Enabled).
			SetServerURL(cfg.ServerURL).
			SetTopic(cfg.Topic).
			SetToken(cfg.Token).
			SetOnTemplate(cfg.OnTemplate).
			SetOffTemplate(cfg.OffTemplate).
			Save(ctx)
	} else {
		_, err = existing.Update().
			SetEnabled(cfg.Enabled).
			SetServerURL(cfg.ServerURL).
			SetTopic(cfg.Topic).
			SetToken(cfg.Token).
			SetOnTemplate(cfg.OnTemplate).
			SetOffTemplate(cfg.OffTemplate).
			Save(ctx)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("保存配置失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "配置已保存"})
}

func TestNtfyNotification(c *gin.Context) {
	if !requireScheduleUseCase(c) {
		return
	}

	ctx := context.Background()
	client := scheduleUseCase.GetClient()
	config, err := client.NtfyConfig.Query().First(ctx)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ntfy未配置"})
		return
	}

	cfg := ntfy.Config{
		Enabled:     config.Enabled,
		ServerURL:   config.ServerURL,
		Topic:       config.Topic,
		Token:       config.Token,
		OnTemplate:  config.OnTemplate,
		OffTemplate: config.OffTemplate,
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	_ = ctx
	if err := ntfy.TestConnection(cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("测试连接失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "测试通知已发送"})
}

// getNtfyConfigForScheduler retrieves ntfy config as ntfy.Config struct.
func getNtfyConfigForScheduler(ctx context.Context) (*ntfy.Config, error) {
	if scheduleUseCase == nil {
		return ntfy.DefaultConfig(), nil
	}
	client := scheduleUseCase.GetClient()
	config, err := client.NtfyConfig.Query().First(ctx)
	if err != nil {
		return ntfy.DefaultConfig(), err
	}
	return &ntfy.Config{
		Enabled:     config.Enabled,
		ServerURL:   config.ServerURL,
		Topic:       config.Topic,
		Token:       config.Token,
		OnTemplate:  config.OnTemplate,
		OffTemplate: config.OffTemplate,
	}, nil
}

// GetNtfyConfigForScheduler is exported for the scheduler package.
func GetNtfyConfigForScheduler(ctx context.Context) (*ntfy.Config, error) {
	return getNtfyConfigForScheduler(ctx)
}

// getServerChanConfigForScheduler retrieves ServerChan config.
func getServerChanConfigForScheduler(ctx context.Context) (*ent.ServerChanConfig, error) {
	if scheduleUseCase == nil {
		return &ent.ServerChanConfig{
			SendKey:     "",
			OnTemplate:  "{{.Name}} 任务执行成功，灯已开启",
			OffTemplate: "{{.Name}} 任务执行成功，灯已关闭",
			Enabled:     false,
		}, nil
	}
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

// GetServerChanConfigForScheduler is exported for the scheduler package.
func GetServerChanConfigForScheduler(ctx context.Context) (*ent.ServerChanConfig, error) {
	return getServerChanConfigForScheduler(ctx)
}
