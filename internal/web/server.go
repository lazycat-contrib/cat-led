package web

import (
	"cat-led/internal/handlers"
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed public
var staticFS embed.FS

// Server 表示Web服务器
type Server struct {
	engine *gin.Engine
}

// NewServer 创建一个新的Web服务器
func NewServer() *Server {
	r := gin.Default()
	return &Server{
		engine: r,
	}
}

// SetupRoutes 设置路由和静态文件
func (s *Server) SetupRoutes() error {
	// 提取嵌入的静态文件到子文件系统
	publicFS, err := fs.Sub(staticFS, "public")
	if err != nil {
		return err
	}

	// 设置静态文件服务
	s.engine.StaticFS("/static", http.FS(publicFS))

	// 使用嵌入式HTML模板
	templ := template.New("html")
	templ, err = templ.ParseFS(staticFS, "public/index.html")
	if err != nil {
		return err
	}
	s.engine.SetHTMLTemplate(templ)

	// HTML文件路由
	s.engine.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})

	// API路由
	s.engine.GET("/reboot", handlers.Reboot)
	s.engine.GET("/ledcontrol", handlers.LedControl)
	s.engine.GET("/api/led-status", handlers.GetLedStatus)
	s.engine.GET("/userinfo", handlers.GetUserInfo)

	// 定时任务相关API
	s.engine.GET("/api/schedules", handlers.GetSchedules)
	s.engine.POST("/api/schedules", handlers.CreateSchedule)
	s.engine.PUT("/api/schedules/:id", handlers.UpdateSchedule)
	s.engine.DELETE("/api/schedules/:id", handlers.DeleteSchedule)

	return nil
}

// Run 启动Web服务器
func (s *Server) Run(addr string) error {
	log.Printf("Starting server at %s", addr)
	return s.engine.Run(addr)
}
