package web

import (
	"cat-led/internal/auth"
	"cat-led/internal/handlers"
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

//go:embed public
var staticFS embed.FS

// Server 表示Web服务器
type Server struct {
	engine       *gin.Engine
	oidcProvider *auth.OIDCProvider
}

// NewServer 创建一个新的Web服务器
func NewServer() (*Server, error) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// 初始化OIDC提供者
	oidcProvider, err := auth.NewOIDCProvider()
	if err != nil {
		log.Printf("Warning: Failed to initialize OIDC provider: %v", err)
		// OIDC不可用时仍然可以启动服务器，但需要依赖Header认证
		oidcProvider = nil
	}

	return &Server{
		engine:       r,
		oidcProvider: oidcProvider,
	}, nil
}

// SetupRoutes 设置路由和静态文件
func (s *Server) SetupRoutes() error {
	// 提取嵌入的静态文件到子文件系统
	publicFS, err := fs.Sub(staticFS, "public")
	if err != nil {
		return err
	}

	// 使用嵌入式HTML模板
	tpl := template.New("html")
	tpl, err = tpl.ParseFS(staticFS, "public/index.html", "public/config.html", "public/login.html")
	if err != nil {
		return err
	}
	s.engine.SetHTMLTemplate(tpl)

	// 添加会话中间件
	s.engine.Use(auth.SessionMiddleware())

	// 登录相关路由（不需要认证）
	s.engine.GET("/login", func(c *gin.Context) {
		// 构建模板数据
		data := gin.H{
			"OIDCAvailable": false,
		}

		// 检查OIDC是否可用
		if s.oidcProvider != nil {
			data["OIDCAvailable"] = true

			// 构建登录URL
			oidcBasePath := os.Getenv("LAZYCAT_AUTH_OIDC_BASE_PATH")
			if oidcBasePath == "" {
				oidcBasePath = "/auth/oidc"
			}
			data["OIDCLoginURL"] = oidcBasePath + "/login"
		}

		c.HTML(200, "login.html", data)
	})

	if s.oidcProvider != nil {
		// 从环境变量获取OIDC路径，默认为 /auth/oidc
		oidcBasePath := os.Getenv("LAZYCAT_AUTH_OIDC_BASE_PATH")
		if oidcBasePath == "" {
			oidcBasePath = "/auth/oidc"
		}

		callbackPath := os.Getenv("LAZYCAT_AUTH_OIDC_CALLBACK_PATH")
		if callbackPath == "" {
			callbackPath = "/auth/oidc/callback"
		}

		s.engine.GET(oidcBasePath+"/login", s.oidcProvider.HandleLogin)
		s.engine.GET(callbackPath, s.oidcProvider.HandleCallback)
	}

	s.engine.GET("/logout", auth.HandleLogout)

	// 设置静态文件服务（不需要认证）
	s.engine.StaticFS("/static", http.FS(publicFS))

	// 需要认证的路由组
	authenticated := s.engine.Group("/")
	// 总是应用认证中间件，即使OIDC Provider为nil也要检查Header认证
	authenticated.Use(auth.AuthMiddleware(s.oidcProvider))

	// HTML文件路由
	authenticated.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})
	authenticated.GET("/config.html", func(c *gin.Context) {
		c.HTML(200, "config.html", nil)
	})

	// API路由
	authenticated.GET("/ledcontrol", handlers.LedControl)
	authenticated.GET("/api/led-status", handlers.GetLedStatus)
	authenticated.GET("/userinfo", handlers.GetUserInfo)

	// 定时任务相关API
	authenticated.GET("/api/schedules", handlers.GetSchedules)
	authenticated.POST("/api/schedules", handlers.CreateSchedule)
	authenticated.PUT("/api/schedules/:id", handlers.UpdateSchedule)
	authenticated.DELETE("/api/schedules/:id", handlers.DeleteSchedule)

	// Server酱配置相关API
	authenticated.GET("/api/serverchan/config", handlers.GetServerChanConfig)
	authenticated.POST("/api/serverchan/config", handlers.SaveServerChanConfig)

	return nil
}

// Run 启动Web服务器
func (s *Server) Run(addr string) error {
	log.Printf("Starting server at %s", addr)
	return s.engine.Run(addr)
}
