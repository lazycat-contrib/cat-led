package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

// Environment variable names for OIDC configuration.
const (
	envOIDCClientID     = "LAZYCAT_AUTH_OIDC_CLIENT_ID"
	envOIDCClientSecret = "LAZYCAT_AUTH_OIDC_CLIENT_SECRET"
	envOIDCAuthURI      = "LAZYCAT_AUTH_OIDC_AUTH_URI"
	envOIDCTokenURI     = "LAZYCAT_AUTH_OIDC_TOKEN_URI"
	envOIDCUserInfoURI  = "LAZYCAT_AUTH_OIDC_USERINFO_URI"
	envOIDCRedirectURL  = "LAZYCAT_AUTH_OIDC_REDIRECT_URL"
	envOIDCBasePath     = "LAZYCAT_AUTH_OIDC_BASE_PATH"
	envOIDCCallbackPath = "LAZYCAT_AUTH_OIDC_CALLBACK_PATH"
	envAppDomain        = "LAZYCAT_APP_DOMAIN"
)

// Default paths for OIDC endpoints.
const (
	defaultOIDCBasePath     = "/auth/oidc"
	defaultOIDCCallbackPath = "/auth/oidc/callback"
	defaultAppDomain        = "localhost:3000"
)

// Cookie configuration.
const (
	cookieUserID    = "user_id"
	cookieUserRole  = "user_role"
	cookieOIDCState = "oidc_state"
	cookieMaxAge    = 3600 * 24 // 24 hours
	stateMaxAge     = 300       // 5 minutes
)

// User roles.
const (
	roleUser  = "USER"
	roleAdmin = "ADMIN"
)

// OIDCConfig holds the OIDC provider configuration.
type OIDCConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	AuthURL      string
	TokenURL     string
	UserInfoURL  string
}

// OIDCProvider manages OIDC authentication.
type OIDCProvider struct {
	config      *oauth2.Config
	userInfoURL string
}

// NewOIDCProvider creates a new OIDC provider from environment variables.
func NewOIDCProvider() (*OIDCProvider, error) {
	config, err := loadOIDCConfig()
	if err != nil {
		return nil, err
	}

	log.Printf("OIDC client ID: %s", config.ClientID)
	log.Printf("OIDC client secret set: %v", config.ClientSecret != "")

	return createOIDCProvider(config)
}

// loadOIDCConfig reads OIDC configuration from environment variables.
func loadOIDCConfig() (*OIDCConfig, error) {
	clientID := os.Getenv(envOIDCClientID)
	clientSecret := os.Getenv(envOIDCClientSecret)
	authURL := os.Getenv(envOIDCAuthURI)
	tokenURL := os.Getenv(envOIDCTokenURI)
	userInfoURL := os.Getenv(envOIDCUserInfoURI)

	if clientID == "" || clientSecret == "" || authURL == "" || tokenURL == "" {
		return nil, errors.New("missing required OIDC configuration")
	}

	return &OIDCConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  buildRedirectURL(),
		AuthURL:      authURL,
		TokenURL:     tokenURL,
		UserInfoURL:  userInfoURL,
	}, nil
}

// createOIDCProvider creates an OIDCProvider from the given configuration.
func createOIDCProvider(config *OIDCConfig) (*OIDCProvider, error) {
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
		config:      oauth2Config,
		userInfoURL: config.UserInfoURL,
	}, nil
}

// buildRedirectURL constructs the OAuth2 redirect URL.
func buildRedirectURL() string {
	if redirectURL := os.Getenv(envOIDCRedirectURL); redirectURL != "" {
		return redirectURL
	}

	domain := getEnvOrDefault(envAppDomain, defaultAppDomain)
	callbackPath := getEnvOrDefault(envOIDCCallbackPath, defaultOIDCCallbackPath)

	return fmt.Sprintf("https://%s%s", domain, callbackPath)
}

// getEnvOrDefault returns the environment variable value or a default.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetOIDCBasePath returns the configured OIDC base path.
func GetOIDCBasePath() string {
	return getEnvOrDefault(envOIDCBasePath, defaultOIDCBasePath)
}

// GetOIDCCallbackPath returns the configured OIDC callback path.
func GetOIDCCallbackPath() string {
	return getEnvOrDefault(envOIDCCallbackPath, defaultOIDCCallbackPath)
}

// generateState generates a cryptographically random state string for CSRF protection.
func generateState() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// AuthMiddleware checks for valid authentication via header or session.
func AuthMiddleware(oidcProvider *OIDCProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for authentication header (injected by system)
		if userID := c.GetHeader("x-hc-user-id"); userID != "" {
			c.Next()
			return
		}

		// Check for session-based authentication
		if sessionUserID, exists := c.Get("user_id"); exists && sessionUserID != "" {
			c.Header("x-hc-user-id", sessionUserID.(string))
			if userRole, exists := c.Get("user_role"); exists {
				c.Header("x-hc-user-role", userRole.(string))
			}
			c.Next()
			return
		}

		// Allow access to public paths
		if isPublicPath(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Redirect to login
		redirectToLogin(c, oidcProvider)
	}
}

// isPublicPath checks if the path should be accessible without authentication.
func isPublicPath(path string) bool {
	oidcBasePath := GetOIDCBasePath()
	return strings.HasPrefix(path, "/login") ||
		strings.HasPrefix(path, oidcBasePath) ||
		strings.HasPrefix(path, "/static")
}

// redirectToLogin sends the user to the login page.
func redirectToLogin(c *gin.Context, oidcProvider *OIDCProvider) {
	if oidcProvider != nil {
		c.Redirect(http.StatusFound, "/login")
	} else {
		c.Redirect(http.StatusFound, "/login?error=oidc_not_configured")
	}
	c.Abort()
}

// HandleLogin initiates the OIDC login flow.
func (p *OIDCProvider) HandleLogin(c *gin.Context) {
	state := generateState()
	c.SetCookie(cookieOIDCState, state, stateMaxAge, "/", "", false, true)

	authURL := p.config.AuthCodeURL(state)
	c.Redirect(http.StatusFound, authURL)
}

// HandleCallback processes the OIDC callback after authentication.
func (p *OIDCProvider) HandleCallback(c *gin.Context) {
	if err := p.validateState(c); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Clear state cookie
	c.SetCookie(cookieOIDCState, "", -1, "/", "", false, true)

	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing authorization code"})
		return
	}

	ctx := context.Background()
	token, err := p.config.Exchange(ctx, code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange token"})
		return
	}

	userInfo, err := p.fetchUserInfo(ctx, token)
	if err != nil {
		log.Printf("Warning: Failed to fetch user info: %v", err)
	}

	userID, userRole := extractUserIdentity(userInfo)
	if userID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user ID"})
		return
	}

	// Set session cookies
	c.SetCookie(cookieUserID, userID, cookieMaxAge, "/", "", false, true)
	c.SetCookie(cookieUserRole, userRole, cookieMaxAge, "/", "", false, true)

	c.Redirect(http.StatusFound, "/")
}

// validateState verifies the OIDC state parameter matches the cookie.
func (p *OIDCProvider) validateState(c *gin.Context) error {
	expectedState, err := c.Cookie(cookieOIDCState)
	if err != nil {
		return errors.New("missing state cookie")
	}

	if c.Query("state") != expectedState {
		return errors.New("invalid state")
	}

	return nil
}

// fetchUserInfo retrieves user information from the OIDC provider.
func (p *OIDCProvider) fetchUserInfo(ctx context.Context, token *oauth2.Token) (map[string]interface{}, error) {
	if p.userInfoURL == "" {
		return nil, errors.New("user info URL not configured")
	}

	client := p.config.Client(ctx, token)
	resp, err := client.Get(p.userInfoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer resp.Body.Close()

	var userInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return userInfo, nil
}

// extractUserIdentity extracts the user ID and role from the user info response.
func extractUserIdentity(userInfo map[string]interface{}) (userID, userRole string) {
	userRole = roleUser

	if userInfo == nil {
		return "", userRole
	}

	// Try to get user ID from various claims
	if sub, ok := userInfo["sub"].(string); ok {
		userID = sub
	} else if preferredUsername, ok := userInfo["preferred_username"].(string); ok {
		userID = preferredUsername
	}

	// Check for admin role in groups
	if groups, ok := userInfo["groups"].([]interface{}); ok {
		for _, group := range groups {
			if groupStr, ok := group.(string); ok && groupStr == roleAdmin {
				userRole = roleAdmin
				break
			}
		}
	}

	return userID, userRole
}

// HandleLogout clears session cookies and redirects to login.
func HandleLogout(c *gin.Context) {
	c.SetCookie(cookieUserID, "", -1, "/", "", false, true)
	c.SetCookie(cookieUserRole, "", -1, "/", "", false, true)
	c.Redirect(http.StatusFound, "/login")
}

// SessionMiddleware restores user information from cookies into the request context.
func SessionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if userID, err := c.Cookie(cookieUserID); err == nil && userID != "" {
			c.Set("user_id", userID)
			if userRole, err := c.Cookie(cookieUserRole); err == nil && userRole != "" {
				c.Set("user_role", userRole)
			}
		}
		c.Next()
	}
}
