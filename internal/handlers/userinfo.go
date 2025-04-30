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

	currentUser := BasicInfo{
		UserId:        c.GetHeader("x-hc-user-id"),
		UserRole:      c.GetHeader("x-hc-user-role"),
		DeviceID:      c.GetHeader("x-hc-device-id"),
		DeviceVersion: c.GetHeader("x-hc-device-version"),
	}
	catUser := LazyCatUser{
		BasicInfo: currentUser,
	}
	userInfo, err := gw.Users.QueryUserInfo(ctx, &users.UserID{Uid: currentUser.UserId})
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	catUser.Detail = userInfo
	c.JSON(200, catUser)
}
