package handlers

import (
	gohelper "gitee.com/linakesi/lzc-sdk/lang/go"
	users "gitee.com/linakesi/lzc-sdk/lang/go/common"
	"github.com/gin-gonic/gin"
	"os"
)

type LoginInfo struct {
	DeviceID      string
	DeviceVersion string
	UserId        string
	UserRole      string
}

func listAllUers() []string {
	var ret []string
	dirs, err := os.ReadDir("/lzcapp/run/mnt/home")
	if err != nil {
		return nil
	}
	for _, d := range dirs {
		ret = append(ret, d.Name())
	}
	return ret
}

func GetUserInfo(c *gin.Context) {
	ctx := c.Request.Context()
	gw, err := gohelper.NewAPIGateway(ctx)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	defer gw.Close()
	var ret struct {
		CurrentUserInfo LoginInfo
		Detail          *users.UserInfo
	}
	currentUser := LoginInfo{
		UserId:        c.GetHeader("x-hc-user-id"),
		UserRole:      c.GetHeader("x-hc-user-role"),
		DeviceID:      c.GetHeader("x-hc-device-id"),
		DeviceVersion: c.GetHeader("x-hc-device-version"),
	}

	userInfo, err := gw.Users.QueryUserInfo(ctx, &users.UserID{Uid: currentUser.UserId})
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	ret.Detail = userInfo
	c.JSON(200, ret)
}
