package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cat-led/internal/handlers"
	"cat-led/internal/pkg/zlog"
	"cat-led/internal/web"
)

// Default configuration values.
const (
	defaultPort    = "3000"
	defaultDBPath  = "/lzcapp/data/schedules.db"
	defaultLogDir  = "/lzcapp/data/logs"
	defaultLogFile = "cat-led.log"
)

// Log configuration defaults.
const (
	logMaxSize    = 10 // MB
	logMaxBackups = 5
	logMaxAge     = 30 // days
)

// Shutdown timeout.
const shutdownTimeout = 5 * time.Second

func main() {
	logger := initLogger()
	logger.Info().Msg("懒猫关灯助手启动中...")

	initializeServices(logger)

	server := createServer(logger)

	port := getEnvOrDefault("PORT", defaultPort)
	go startServer(server, port, logger)

	waitForShutdown(logger)
}

// initLogger initializes the logging system.
func initLogger() *zlog.Logger {
	logLevel := getEnvOrDefault("LOG_LEVEL", "info")
	logDir := getEnvOrDefault("LOG_DIR", defaultLogDir)
	logFile := getEnvOrDefault("LOG_FILE", defaultLogFile)

	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("警告: 无法创建日志目录 %s: %v", logDir, err)
	}

	return zlog.NewLogger(zlog.LogConfig{
		LogLevel:    logLevel,
		LogDir:      logDir,
		LogFileName: logFile,
		MaxSize:     logMaxSize,
		MaxBackups:  logMaxBackups,
		MaxAge:      logMaxAge,
	})
}

// initializeServices initializes the database, LED status, and scheduler.
func initializeServices(logger *zlog.Logger) {
	dbPath := getEnvOrDefault("DB_PATH", defaultDBPath)
	handlers.InitScheduleUseCase(dbPath, logger)
	logger.Info().Str("db_path", dbPath).Msg("数据库初始化完成")

	ctx := context.Background()
	handlers.InitLedStatus(ctx, logger)
	logger.Info().Msg("LED状态初始化完成")

	handlers.InitScheduler(logger)
	logger.Info().Msg("定时任务调度器已启动")
}

// createServer creates and configures the web server.
func createServer(logger *zlog.Logger) *web.Server {
	server, err := web.NewServer()
	if err != nil {
		logger.Fatal().Err(err).Msg("无法创建Web服务器")
	}

	if err := server.SetupRoutes(); err != nil {
		logger.Fatal().Err(err).Msg("无法设置路由")
	}

	return server
}

// startServer starts the web server in a goroutine.
func startServer(server *web.Server, port string, logger *zlog.Logger) {
	addr := fmt.Sprintf(":%s", port)
	logger.Info().Str("port", port).Msg("准备启动Web服务器")

	if err := server.Run(addr); err != nil {
		logger.Fatal().Err(err).Msg("Web服务器启动失败")
	}
}

// waitForShutdown waits for termination signals and performs graceful shutdown.
func waitForShutdown(logger *zlog.Logger) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("收到关闭信号，正在优雅关闭...")

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	<-ctx.Done()

	logger.Info().Msg("懒猫关灯助手已关闭")
}

// getEnvOrDefault returns the environment variable value or a default.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
