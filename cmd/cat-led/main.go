package main

import (
	"cat-led/internal/handlers"
	"cat-led/internal/pkg/zlog"
	"cat-led/internal/web"
	"context"
)

func main() {

	logConfig := zlog.LogConfig{
		LogLevel:    "info",
		LogDir:      "/lzcapp/var/logs",
		LogFileName: "app.log",
		MaxSize:     10, // 10 MB
		MaxBackups:  5,  // 保留5个备份文件
		MaxAge:      7,  // 保留7天的日志文件
	}

	logger := zlog.NewLogger(logConfig)
	// 确定数据库路径
	dbPath := getDbPath()

	// 初始化scheduleUseCase
	handlers.InitScheduleUseCase(dbPath, logger)

	// 初始化LED状态
	handlers.InitLedStatus(context.Background(), logger)

	// 初始化定时任务调度器
	handlers.InitScheduler(logger)

	// 创建Web服务器
	server := web.NewServer()

	// 设置路由和静态文件
	if err := server.SetupRoutes(); err != nil {
		logger.Fatal().Err(err).Msg("设置路由失败")
	}

	// 启动服务器
	logger.Info().Msg("Starting server at :3000")
	if err := server.Run(":3000"); err != nil {
		logger.Fatal().Err(err).Msg("启动服务器失败")
	}
}

// getDbPath 获取数据库文件路径
func getDbPath() string {
	// 首选标准数据目录
	dbPath := "/lzcapp/var/cat_led.db"

	return dbPath
}
