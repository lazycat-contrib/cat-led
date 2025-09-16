package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

type OIDCConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	AuthURL      string
	TokenURL     string
	UserInfoURL  string
}

type OIDCProvider struct {
	config *oauth2.Config
}

// NewOIDCProvider 创建OIDC提供者
func NewOIDCProvider() (*OIDCProvider, error) {
	// 从环境变量获取配置
	clientID := os.Getenv("LAZYCAT_AUTH_OIDC_CLIENT_ID")
	clientSecret := os.Getenv("LAZYCAT_AUTH_OIDC_CLIENT_SECRET")
	authURL := os.Getenv("LAZYCAT_AUTH_OIDC_AUTH_URI")
	tokenURL := os.Getenv("LAZYCAT_AUTH_OIDC_TOKEN_URI")
	userInfoURL := os.Getenv("LAZYCAT_AUTH_OIDC_USERINFO_URI")

	// 检查必需的配置
	if clientID == "" || clientSecret == "" || authURL == "" || tokenURL == "" {
		return nil, fmt.Errorf("missing required OIDC configuration")
	}

	log.Printf("OIDC client ID: %s", clientID)
	log.Printf("OIDC client secret set: %v", clientSecret != "")

	config := &OIDCConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  getRedirectURL(),
		AuthURL:      authURL,
		TokenURL:     tokenURL,
		UserInfoURL:  userInfoURL,
	}

	return createOIDCProvider(config)
}

// createOIDCProvider 创建OIDC Provider实例
func createOIDCProvider(config *OIDCConfig) (*OIDCProvider, error) {
	// 配置OAuth2
	oauth2Config := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "groups"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  config.AuthURL,
			TokenURL: config.TokenURL,
		},
	}

	return &OIDCProvider{
		config: oauth2Config,
	}, nil
}

// getRedirectURL 获取重定向URL
func getRedirectURL() string {
	// 优先从环境变量获取完整的回调URL
	redirectURL := os.Getenv("LAZYCAT_AUTH_OIDC_REDIRECT_URL")
	if redirectURL != "" {
		return redirectURL
	}

	// 使用应用分配的域名
	domain := os.Getenv("LAZYCAT_APP_DOMAIN")
	if domain == "" {
		domain = "localhost:3000" // 开发环境默认值
	}

	// 从环境变量获取回调路径，默认为 /auth/oidc/callback
	callbackPath := os.Getenv("LAZYCAT_AUTH_OIDC_CALLBACK_PATH")
	if callbackPath == "" {
		callbackPath = "/auth/oidc/callback"
	}

	return fmt.Sprintf("https://%s%s", domain, callbackPath)
}

// generateState 生成随机状态字符串
func generateState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// AuthMiddleware OIDC认证中间件
func AuthMiddleware(oidcProvider *OIDCProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查是否已有认证Header
		userID := c.GetHeader("x-hc-user-id")
		if userID != "" {
			// 已有认证信息，继续处理
			c.Next()
			return
		}

		// 检查是否有会话中的用户信息
		sessionUserID, exists := c.Get("user_id")
		if exists && sessionUserID != "" {
			// 设置Header供后续处理使用
			c.Header("x-hc-user-id", sessionUserID.(string))
			if userRole, exists := c.Get("user_role"); exists {
				c.Header("x-hc-user-role", userRole.(string))
			}
			c.Next()
			return
		}

		// 如果是登录页面或回调页面，允许访问
		oidcBasePath := os.Getenv("LAZYCAT_AUTH_OIDC_BASE_PATH")
		if oidcBasePath == "" {
			oidcBasePath = "/auth/oidc"
		}

		if strings.HasPrefix(c.Request.URL.Path, "/login") ||
			strings.HasPrefix(c.Request.URL.Path, oidcBasePath) ||
			strings.HasPrefix(c.Request.URL.Path, "/static") {
			c.Next()
			return
		}

		// 没有认证信息，重定向到登录页面
		if oidcProvider != nil {
			c.Redirect(http.StatusFound, "/login")
		} else {
			c.Redirect(http.StatusFound, "/login?error=oidc_not_configured")
		}
		c.Abort()
	}
}

// HandleLogin 处理登录请求
func (p *OIDCProvider) HandleLogin(c *gin.Context) {
	state := generateState()

	// 存储state到session或cookie中（这里简化处理，实际应用中应该使用安全的session管理）
	c.SetCookie("oidc_state", state, 300, "/", "", false, true)

	authURL := p.config.AuthCodeURL(state)
	c.Redirect(http.StatusFound, authURL)
}

// HandleCallback 处理OIDC回调
func (p *OIDCProvider) HandleCallback(c *gin.Context) {
	// 验证state
	expectedState, err := c.Cookie("oidc_state")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing state cookie"})
		return
	}

	if c.Query("state") != expectedState {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid state"})
		return
	}

	// 清除state cookie
	c.SetCookie("oidc_state", "", -1, "/", "", false, true)

	// 获取授权码
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing authorization code"})
		return
	}

	// 交换token
	ctx := context.Background()
	token, err := p.config.Exchange(ctx, code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange token"})
		return
	}

	// 从UserInfo端点获取用户信息
	var userInfo map[string]interface{}
	userInfoURL := os.Getenv("LAZYCAT_AUTH_OIDC_USERINFO_URI")
	if userInfoURL != "" {
		client := p.config.Client(ctx, token)
		resp, err := client.Get(userInfoURL)
		if err == nil {
			defer resp.Body.Close()
			if err := json.NewDecoder(resp.Body).Decode(&userInfo); err == nil {
				// 成功获取用户信息
			}
		}
	}

	// 提取用户信息
	userID := ""
	userRole := "USER"

	if sub, ok := userInfo["sub"].(string); ok {
		userID = sub
	} else if preferred_username, ok := userInfo["preferred_username"].(string); ok {
		userID = preferred_username
	}

	// 检查用户组/角色
	if groups, ok := userInfo["groups"].([]interface{}); ok {
		for _, group := range groups {
			if groupStr, ok := group.(string); ok && groupStr == "ADMIN" {
				userRole = "ADMIN"
				break
			}
		}
	}

	if userID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user ID"})
		return
	}

	// 设置会话信息（这里简化处理，实际应用中应该使用安全的session管理）
	c.SetCookie("user_id", userID, 3600*24, "/", "", false, true)
	c.SetCookie("user_role", userRole, 3600*24, "/", "", false, true)

	// 重定向到首页
	c.Redirect(http.StatusFound, "/")
}

// HandleLogout 处理登出
func HandleLogout(c *gin.Context) {
	// 清除cookies
	c.SetCookie("user_id", "", -1, "/", "", false, true)
	c.SetCookie("user_role", "", -1, "/", "", false, true)

	// 重定向到登录页面
	c.Redirect(http.StatusFound, "/login")
}

// SessionMiddleware 会话中间件，从cookie中恢复用户信息
func SessionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if userID, err := c.Cookie("user_id"); err == nil && userID != "" {
			c.Set("user_id", userID)
			if userRole, err := c.Cookie("user_role"); err == nil && userRole != "" {
				c.Set("user_role", userRole)
			}
		}
		c.Next()
	}
}
