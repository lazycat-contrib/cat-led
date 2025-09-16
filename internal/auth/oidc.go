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
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

type OIDCConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Issuer       string
	AuthURL      string
	TokenURL     string
	UserInfoURL  string
}

// WellKnownConfig OIDC well-known配置结构
type WellKnownConfig struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	UserInfoEndpoint                  string   `json:"userinfo_endpoint"`
	JwksURI                           string   `json:"jwks_uri"`
	ScopesSupported                   []string `json:"scopes_supported"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	GrantTypesSupported               []string `json:"grant_types_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
	SubjectTypesSupported             []string `json:"subject_types_supported"`
	IdTokenSigningAlgValuesSupported  []string `json:"id_token_signing_alg_values_supported"`
	ClaimsSupported                   []string `json:"claims_supported"`
}

type OIDCProvider struct {
	config   *oauth2.Config
	verifier *oidc.IDTokenVerifier
	provider *oidc.Provider

	// 延迟初始化相关
	initialized bool
	initError   error
}

// LazyOIDCProvider 延迟初始化的OIDC Provider
type LazyOIDCProvider struct {
	actualProvider *OIDCProvider
	initialized    bool
	initError      error
}

// discoverOIDCConfig 动态发现OIDC配置
func discoverOIDCConfig(host string) (*WellKnownConfig, error) {
	if host == "" {
		return nil, fmt.Errorf("host is required for OIDC discovery")
	}

	log.Printf("Discovering OIDC config for host: %s", host)

	// 解析域名，提取微服名称
	// 假设域名格式为: lazy-cat-led-helper.tinycat.heiyu.space
	parts := strings.Split(host, ".")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid domain format: %s", host)
	}

	// 提取微服名称（第二个部分）
	microServiceName := parts[1] // 例如: tinycat
	log.Printf("Extracted microservice name: %s", microServiceName)

	// 构建well-known配置URL
	wellKnownURL := fmt.Sprintf("https://%s.heiyu.space/sys/oauth/.well-known/openid-configuration", microServiceName)
	log.Printf("Fetching OIDC discovery from: %s", wellKnownURL)

	// 创建HTTP客户端，设置超时
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 发起请求
	resp, err := client.Get(wellKnownURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch OIDC discovery: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OIDC discovery returned status: %d", resp.StatusCode)
	}

	// 解析JSON响应
	var config WellKnownConfig
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to parse OIDC discovery response: %v", err)
	}

	log.Printf("OIDC discovery successful - Issuer: %s", config.Issuer)
	log.Printf("Supported grant types: %v", config.GrantTypesSupported)
	log.Printf("Supported token endpoint auth methods: %v", config.TokenEndpointAuthMethodsSupported)

	return &config, nil
}

// determineClientConfig 根据OIDC发现信息确定客户端配置
func determineClientConfig(wellKnownConfig *WellKnownConfig, host string) (clientID, clientSecret string) {
	// 优先使用环境变量
	if envClientID := os.Getenv("LAZYCAT_AUTH_OIDC_CLIENT_ID"); envClientID != "" {
		clientID = envClientID
	}
	if envClientSecret := os.Getenv("LAZYCAT_AUTH_OIDC_CLIENT_SECRET"); envClientSecret != "" {
		clientSecret = envClientSecret
	}

	// 如果环境变量中已经有完整配置，直接返回
	if clientID != "" {
		return clientID, clientSecret
	}

	// 检查支持的认证方法
	supportsClientSecretPost := false
	supportsClientSecretBasic := false
	for _, method := range wellKnownConfig.TokenEndpointAuthMethodsSupported {
		switch method {
		case "client_secret_post":
			supportsClientSecretPost = true
		case "client_secret_basic":
			supportsClientSecretBasic = true
		}
	}

	// 根据域名提取应用名称作为客户端ID
	parts := strings.Split(host, ".")
	if len(parts) > 0 {
		appName := parts[0] // lazy-cat-led-helper
		clientID = appName
	} else {
		clientID = "default-client"
	}

	// 如果支持客户端密钥认证，但我们没有密钥，可能需要使用公共客户端
	if (supportsClientSecretPost || supportsClientSecretBasic) && clientSecret == "" {
		// 尝试一些常见的公共客户端名称
		possibleClientIDs := []string{
			clientID,
			clientID + "-public",
			"public",
			"webapp",
			"frontend",
		}

		// 这里我们返回第一个尝试的ID，实际使用时可能需要进一步验证
		clientID = possibleClientIDs[0]
	}

	log.Printf("Determined client ID: %s, has secret: %v", clientID, clientSecret != "")

	return clientID, clientSecret
}

// NewOIDCProvider 创建OIDC提供者（现在返回延迟初始化的provider）
func NewOIDCProvider() (*LazyOIDCProvider, error) {
	// 优先从环境变量获取完整配置
	clientID := os.Getenv("LAZYCAT_AUTH_OIDC_CLIENT_ID")
	clientSecret := os.Getenv("LAZYCAT_AUTH_OIDC_CLIENT_SECRET")

	log.Printf("OIDC client ID from env: %s", clientID)
	log.Printf("OIDC client secret set: %v", clientSecret != "")

	// 如果有完整的环境变量配置，立即初始化
	if clientID != "" && clientSecret != "" {
		log.Printf("Using manual OIDC configuration")

		config := &OIDCConfig{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  getRedirectURL(""),
			Issuer:       os.Getenv("LAZYCAT_AUTH_OIDC_ISSUER"),
			AuthURL:      os.Getenv("LAZYCAT_AUTH_OIDC_AUTH_URI"),
			TokenURL:     os.Getenv("LAZYCAT_AUTH_OIDC_TOKEN_URI"),
			UserInfoURL:  os.Getenv("LAZYCAT_AUTH_OIDC_USERINFO_URI"),
		}

		actualProvider, err := createOIDCProvider(config)
		if err != nil {
			return nil, err
		}

		return &LazyOIDCProvider{
			actualProvider: actualProvider,
			initialized:    true,
		}, nil
	}

	// 否则返回延迟初始化的provider
	log.Printf("No manual OIDC config found, will attempt dynamic discovery on first request")

	return &LazyOIDCProvider{
		initialized: false,
	}, nil
}

// initializeWithHost 根据host初始化OIDC provider
func (lazy *LazyOIDCProvider) initializeWithHost(host string) error {
	if lazy.initialized {
		return lazy.initError
	}

	log.Printf("Initializing OIDC provider with host: %s", host)

	// 尝试动态发现OIDC配置
	wellKnownConfig, err := discoverOIDCConfig(host)
	if err != nil {
		lazy.initError = fmt.Errorf("failed to discover OIDC config: %v", err)
		lazy.initialized = true
		return lazy.initError
	}

	// 使用智能客户端配置确定
	clientID, clientSecret := determineClientConfig(wellKnownConfig, host)

	log.Printf("Using dynamic discovery with client ID: %s", clientID)

	// 构建配置
	config := &OIDCConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  getRedirectURL(host),
		Issuer:       wellKnownConfig.Issuer,
		AuthURL:      wellKnownConfig.AuthorizationEndpoint,
		TokenURL:     wellKnownConfig.TokenEndpoint,
		UserInfoURL:  wellKnownConfig.UserInfoEndpoint,
	}

	actualProvider, err := createOIDCProvider(config)
	if err != nil {
		lazy.initError = err
		lazy.initialized = true
		return err
	}

	lazy.actualProvider = actualProvider
	lazy.initialized = true
	return nil
}

// GetProvider 获取实际的OIDC provider，如果未初始化则根据host初始化（公开方法）
func (lazy *LazyOIDCProvider) GetProvider(host string) (*OIDCProvider, error) {
	return lazy.getProvider(host)
}

// getProvider 获取实际的OIDC provider，如果未初始化则根据host初始化
func (lazy *LazyOIDCProvider) getProvider(host string) (*OIDCProvider, error) {
	if !lazy.initialized {
		if err := lazy.initializeWithHost(host); err != nil {
			return nil, err
		}
	}

	if lazy.initError != nil {
		return nil, lazy.initError
	}

	return lazy.actualProvider, nil
}

// createOIDCProvider 创建OIDC Provider实例
func createOIDCProvider(config *OIDCConfig) (*OIDCProvider, error) {
	ctx := context.Background()
	var provider *oidc.Provider
	var err error

	// 如果有Issuer，使用自动发现
	if config.Issuer != "" {
		provider, err = oidc.NewProvider(ctx, config.Issuer)
		if err != nil {
			return nil, fmt.Errorf("failed to create OIDC provider: %v", err)
		}
	}

	// 配置OAuth2
	oauth2Config := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "groups"},
	}

	// 设置端点
	if provider != nil {
		oauth2Config.Endpoint = provider.Endpoint()
	} else {
		// 手动配置端点
		oauth2Config.Endpoint = oauth2.Endpoint{
			AuthURL:  config.AuthURL,
			TokenURL: config.TokenURL,
		}
	}

	// 创建ID Token验证器
	var verifier *oidc.IDTokenVerifier
	if provider != nil {
		verifier = provider.Verifier(&oidc.Config{ClientID: config.ClientID})
	}

	return &OIDCProvider{
		config:   oauth2Config,
		verifier: verifier,
		provider: provider,
	}, nil
}

// getRedirectURL 获取重定向URL
func getRedirectURL(host string) string {
	// 优先从环境变量获取完整的回调URL
	redirectURL := os.Getenv("LAZYCAT_AUTH_OIDC_REDIRECT_URL")
	if redirectURL != "" {
		return redirectURL
	}

	// 如果没有设置完整URL，则构建
	domain := host
	if domain == "" {
		// 如果没有提供host，尝试从环境变量获取
		domain = os.Getenv("LAZYCAT_BOXDOMAIN")
		if domain == "" {
			domain = "localhost:3000" // 开发环境默认值
		}
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
func AuthMiddleware(oidcProvider *LazyOIDCProvider) gin.HandlerFunc {
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

		// 没有认证信息
		if oidcProvider != nil {
			// 检查OIDC是否可用（尝试初始化）
			host := c.Request.Host
			_, err := oidcProvider.getProvider(host)
			if err != nil {
				// OIDC不可用，重定向到带错误信息的登录页面
				c.Redirect(http.StatusFound, "/login?error=oidc_not_configured")
				c.Abort()
				return
			}

			// 如果有OIDC Provider，重定向到登录页面
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
		} else {
			// 如果没有OIDC Provider，也重定向到登录页面
			// 登录页面会显示相应的错误信息
			c.Redirect(http.StatusFound, "/login?error=oidc_not_configured")
			c.Abort()
		}
	}
}

// HandleLogin 处理登录请求
func (lazy *LazyOIDCProvider) HandleLogin(c *gin.Context) {
	host := c.Request.Host
	provider, err := lazy.getProvider(host)
	if err != nil {
		log.Printf("Failed to get OIDC provider: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OIDC service unavailable"})
		return
	}

	provider.HandleLogin(c)
}

// HandleCallback 处理OIDC回调
func (lazy *LazyOIDCProvider) HandleCallback(c *gin.Context) {
	host := c.Request.Host
	provider, err := lazy.getProvider(host)
	if err != nil {
		log.Printf("Failed to get OIDC provider: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OIDC service unavailable"})
		return
	}

	provider.HandleCallback(c)
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

	// 验证ID Token
	var userInfo map[string]interface{}
	if p.verifier != nil {
		rawIDToken, ok := token.Extra("id_token").(string)
		if ok {
			idToken, err := p.verifier.Verify(ctx, rawIDToken)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify ID token"})
				return
			}

			if err := idToken.Claims(&userInfo); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse claims"})
				return
			}
		}
	}

	// 如果没有ID Token或验证失败，尝试从UserInfo端点获取用户信息
	if len(userInfo) == 0 {
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
