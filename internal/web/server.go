package web

import (
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"

	"cat-led/internal/auth"
	"cat-led/internal/handlers"

	"github.com/gin-gonic/gin"
)

//go:embed public
var staticFS embed.FS

// Server represents the web server.
type Server struct {
	engine       *gin.Engine
	oidcProvider *auth.OIDCProvider
}

// NewServer creates a new web server instance.
func NewServer() (*Server, error) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	oidcProvider, err := auth.NewOIDCProvider()
	if err != nil {
		log.Printf("Warning: Failed to initialize OIDC provider: %v", err)
		oidcProvider = nil
	}

	return &Server{
		engine:       r,
		oidcProvider: oidcProvider,
	}, nil
}

// SetupRoutes configures all routes and static file serving.
func (s *Server) SetupRoutes() error {
	publicFS, err := fs.Sub(staticFS, "public")
	if err != nil {
		return err
	}

	if err := s.setupTemplates(); err != nil {
		return err
	}

	s.engine.Use(auth.SessionMiddleware())

	s.setupPublicRoutes(publicFS)
	s.setupAuthenticatedRoutes()

	return nil
}

// setupTemplates loads HTML templates from embedded files.
func (s *Server) setupTemplates() error {
	tpl := template.New("html")
	tpl, err := tpl.ParseFS(staticFS, "public/index.html", "public/config.html", "public/login.html")
	if err != nil {
		return err
	}
	s.engine.SetHTMLTemplate(tpl)
	return nil
}

// setupPublicRoutes configures routes that don't require authentication.
func (s *Server) setupPublicRoutes(publicFS fs.FS) {
	s.engine.GET("/login", s.handleLoginPage)
	s.engine.GET("/logout", auth.HandleLogout)
	s.engine.StaticFS("/static", http.FS(publicFS))

	if s.oidcProvider != nil {
		oidcBasePath := auth.GetOIDCBasePath()
		callbackPath := auth.GetOIDCCallbackPath()

		s.engine.GET(oidcBasePath+"/login", s.oidcProvider.HandleLogin)
		s.engine.GET(callbackPath, s.oidcProvider.HandleCallback)
	}
}

// handleLoginPage renders the login page with OIDC configuration.
func (s *Server) handleLoginPage(c *gin.Context) {
	data := gin.H{
		"OIDCAvailable": s.oidcProvider != nil,
	}

	if s.oidcProvider != nil {
		data["OIDCLoginURL"] = auth.GetOIDCBasePath() + "/login"
	}

	c.HTML(http.StatusOK, "login.html", data)
}

// setupAuthenticatedRoutes configures routes that require authentication.
func (s *Server) setupAuthenticatedRoutes() {
	authenticated := s.engine.Group("/")
	authenticated.Use(auth.AuthMiddleware(s.oidcProvider))

	// HTML pages
	authenticated.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})
	authenticated.GET("/config.html", func(c *gin.Context) {
		c.HTML(http.StatusOK, "config.html", nil)
	})

	// API routes
	authenticated.GET("/ledcontrol", handlers.LedControl)
	authenticated.GET("/api/led-status", handlers.GetLedStatus)
	authenticated.GET("/userinfo", handlers.GetUserInfo)

	// Schedule API
	authenticated.GET("/api/schedules", handlers.GetSchedules)
	authenticated.POST("/api/schedules", handlers.CreateSchedule)
	authenticated.PUT("/api/schedules/:id", handlers.UpdateSchedule)
	authenticated.DELETE("/api/schedules/:id", handlers.DeleteSchedule)

	// ServerChan API
	authenticated.GET("/api/serverchan/config", handlers.GetServerChanConfig)
	authenticated.POST("/api/serverchan/config", handlers.SaveServerChanConfig)

	// User Preference API
	authenticated.GET("/api/user/preference", handlers.GetUserPreference)
	authenticated.PUT("/api/user/preference", handlers.UpdateUserPreference)
}

// Run starts the web server on the specified address.
func (s *Server) Run(addr string) error {
	log.Printf("Starting server at %s", addr)
	return s.engine.Run(addr)
}
