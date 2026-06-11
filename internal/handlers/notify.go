package handlers

import (
	"context"
	"fmt"
	"time"

	"cat-led/internal/pkg/lzcnotify"

	"github.com/gin-gonic/gin"
)

// TestLzcNotification sends a LazyCat built-in notification to the current user's clients.
func TestLzcNotification(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	sent, err := lzcnotify.NotifyUser(ctx, userID, "懒猫小灯测试通知", "懒猫内置通知已启用", "")
	if err != nil && sent == 0 {
		c.JSON(500, gin.H{"error": fmt.Sprintf("发送测试通知失败: %v", err)})
		return
	}
	if sent == 0 {
		c.JSON(404, gin.H{"error": "没有找到可通知的在线客户端"})
		return
	}
	if err != nil {
		c.JSON(200, gin.H{"message": fmt.Sprintf("测试通知已发送到%d个客户端，部分客户端失败", sent)})
		return
	}

	c.JSON(200, gin.H{"message": fmt.Sprintf("测试通知已发送到%d个客户端", sent)})
}
