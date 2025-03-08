package main

import (
	"cat-led/internal/handlers"
	"cat-led/internal/web"
	"context"
	"log"
)

func main() {
	// 初始化LED状态
	handlers.InitLedStatus(context.Background())

	// 初始化定时任务调度器
	handlers.InitScheduler()

	// 创建Web服务器
	server := web.NewServer()

	// 设置路由和静态文件
	if err := server.SetupRoutes(); err != nil {
		log.Fatalf("设置路由失败: %v", err)
	}

	// 启动服务器
	log.Println("Starting server at :3000")
	if err := server.Run(":3000"); err != nil {
		log.Fatalf("启动服务器失败: %v", err)
	}
}
