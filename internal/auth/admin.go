package auth

import (
	"context"
	"net/http"
	"net/url"
	"time"

	gohelper "gitee.com/linakesi/lzc-sdk/lang/go"
	users "gitee.com/linakesi/lzc-sdk/lang/go/common"
	"github.com/gin-gonic/gin"
)

// IsLazyCatAdmin resolves the current role from LazyCat, never a browser claim.
func IsLazyCatAdmin(ctx context.Context, id string) (bool, error) {
	if id == "" {
		return false, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	gw, err := gohelper.NewAPIGateway(ctx)
	if err != nil {
		return false, err
	}
	defer gw.Close()
	info, err := gw.Users.QueryUserInfo(ctx, &users.UserID{Uid: id})
	return err == nil && info != nil && info.Role == users.Role_ROLE_ADMIN, err
}

func RequireAdmin(check func(context.Context, string) (bool, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := UserID(c)
		if id == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "请先登录懒猫账号"})
			return
		}
		ok, err := check(c.Request.Context(), id)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "暂时无法核实懒猫管理员身份"})
			return
		}
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "仅懒猫管理员可管理定时开关机"})
			return
		}
		c.Next()
	}
}

// RequireSameOrigin guards cookie-authenticated power mutations. JSON plus a
// custom header also prevents cross-site form submissions when Origin is absent.
func RequireSameOrigin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodGet {
			c.Next()
			return
		}
		if c.GetHeader("X-Cat-Led-Request") != "1" || c.GetHeader("Sec-Fetch-Site") == "cross-site" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "不允许跨站修改开关机计划"})
			return
		}
		if origin := c.GetHeader("Origin"); origin != "" {
			u, err := url.Parse(origin)
			if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host != c.Request.Host {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "请求来源不匹配"})
				return
			}
		}
		c.Next()
	}
}
