package handlers

import (
	"net/http"

	gohelper "gitee.com/linakesi/lzc-sdk/lang/go"
	users "gitee.com/linakesi/lzc-sdk/lang/go/common"
	"github.com/gin-gonic/gin"
)

// BasicInfo contains basic user and device information.
type BasicInfo struct {
	DeviceID      string
	DeviceVersion string
	UserId        string
	UserRole      string
}

// LazyCatUser represents a user with basic and detailed information.
type LazyCatUser struct {
	BasicInfo BasicInfo `json:"CurrentUserInfo"`
	Detail    *users.UserInfo
}

// GetUserInfo returns the current user's information.
func GetUserInfo(c *gin.Context) {
	ctx := c.Request.Context()

	gw, err := gohelper.NewAPIGateway(ctx)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer gw.Close()

	basicInfo := extractBasicInfo(c)
	if basicInfo.UserId == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	catUser := LazyCatUser{BasicInfo: basicInfo}

	// Try to get detailed user info (may fail for OIDC users)
	userInfo, err := gw.Users.QueryUserInfo(ctx, &users.UserID{Uid: basicInfo.UserId})
	if err == nil {
		catUser.Detail = userInfo
	}

	c.JSON(http.StatusOK, catUser)
}

// extractBasicInfo extracts user and device info from headers and session.
func extractBasicInfo(c *gin.Context) BasicInfo {
	info := BasicInfo{
		UserId:        c.GetHeader("x-hc-user-id"),
		UserRole:      c.GetHeader("x-hc-user-role"),
		DeviceID:      c.GetHeader("x-hc-device-id"),
		DeviceVersion: c.GetHeader("x-hc-device-version"),
	}

	// Fall back to session if header is empty
	if info.UserId == "" {
		if sessionUserID, exists := c.Get("user_id"); exists {
			info.UserId = sessionUserID.(string)
		}
		if sessionUserRole, exists := c.Get("user_role"); exists {
			info.UserRole = sessionUserRole.(string)
		}
	}

	return info
}
