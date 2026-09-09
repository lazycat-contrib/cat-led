package handlers

import (
	"cat-led/internal/auth"
	"cat-led/internal/power"
	"context"
	gohelper "gitee.com/linakesi/lzc-sdk/lang/go"
	users "gitee.com/linakesi/lzc-sdk/lang/go/common"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"time"
)

func ShutdownForPowerPlan(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	gw, err := gohelper.NewAPIGateway(ctx)
	if err != nil {
		return err
	}
	defer gw.Close()
	_, err = gw.Box.Shutdown(ctx, &users.ShutdownRequest{Action: users.ShutdownRequest_Poweroff})
	return err
}
func RegisterPowerRoutes(r *gin.RouterGroup, m *power.Manager) {
	p := r.Group("/api/power-plan", auth.RequireAdmin(auth.IsLazyCatAdmin), auth.RequireSameOrigin())
	p.Use(func(c *gin.Context) {
		if m == nil {
			c.AbortWithStatusJSON(503, gin.H{"error": "开关机计划服务未初始化"})
			return
		}
		c.Next()
	})
	p.GET("", func(c *gin.Context) {
		status, err := m.Status(c.Request.Context())
		if err != nil {
			c.JSON(500, gin.H{"error": "无法读取开关机计划"})
			return
		}
		c.JSON(200, status)
	})
	p.PUT("", func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
		var spec power.Spec
		if err := c.ShouldBindJSON(&spec); err != nil {
			c.JSON(400, gin.H{"error": "请检查计划中的日期和时间"})
			return
		}
		state, err := m.Save(c.Request.Context(), spec, auth.UserID(c), time.Now())
		if err != nil {
			log.Printf("RTC save by %s: %v", auth.UserID(c), err)
			c.JSON(409, gin.H{"error": err.Error()})
			return
		}
		log.Printf("RTC plan saved by %s: shutdown=%s wake=%s", auth.UserID(c), state.ShutdownAt, state.WakeAt)
		c.JSON(200, state)
	})
	p.DELETE("", func(c *gin.Context) {
		state, err := m.Cancel(c.Request.Context(), time.Now())
		if err != nil {
			c.JSON(409, gin.H{"error": err.Error()})
			return
		}
		log.Printf("RTC plan cancelled by %s", auth.UserID(c))
		c.JSON(200, state)
	})
}
