package handlers

import (
	gohelper "gitee.com/linakesi/lzc-sdk/lang/go"
	users "gitee.com/linakesi/lzc-sdk/lang/go/common"
	"github.com/gin-gonic/gin"
)

type BasicInfo struct {
	DeviceID      string
	DeviceVersion string
	UserId        string
	UserRole      string
}

type LazyCatUser struct {
	BasicInfo BasicInfo `json:"CurrentUserInfo"`
	Detail    *users.UserInfo
}

func GetUserInfo(c *gin.Context) {
	ctx := c.Request.Context()
	gw, err := gohelper.NewAPIGateway(ctx)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	defer gw.Close()

	// 优先从Header获取用户信息（由系统注入）
	userID := c.GetHeader("x-hc-user-id")
	userRole := c.GetHeader("x-hc-user-role")
	deviceID := c.GetHeader("x-hc-device-id")
	deviceVersion := c.GetHeader("x-hc-device-version")

	// 如果Header中没有，尝试从session/context获取（OIDC认证）
	if userID == "" {
		if sessionUserID, exists := c.Get("user_id"); exists {
			userID = sessionUserID.(string)
		}
		if sessionUserRole, exists := c.Get("user_role"); exists {
			userRole = sessionUserRole.(string)
		}
	}

	// 如果仍然没有用户信息，返回错误
	if userID == "" {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	currentUser := BasicInfo{
		UserId:        userID,
		UserRole:      userRole,
		DeviceID:      deviceID,
		DeviceVersion: deviceVersion,
	}

	catUser := LazyCatUser{
		BasicInfo: currentUser,
	}

	// 尝试获取详细用户信息
	userInfo, err := gw.Users.QueryUserInfo(ctx, &users.UserID{Uid: currentUser.UserId})
	if err != nil {
		// 如果获取详细信息失败，仍然返回基础信息
		// 这样可以兼容OIDC认证的用户
		c.JSON(200, catUser)
		return
	}

	catUser.Detail = userInfo
	c.JSON(200, catUser)
}
