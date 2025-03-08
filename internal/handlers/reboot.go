package handlers

import (
	gohelper "gitee.com/linakesi/lzc-sdk/lang/go"
	users "gitee.com/linakesi/lzc-sdk/lang/go/common"
	"github.com/gin-gonic/gin"
)

func Reboot(c *gin.Context) {
	c.String(200, "now reboot lazycat...")
	gw, err := gohelper.NewAPIGateway(c.Request.Context())
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	defer gw.Close()

	gw.Box.Shutdown(c.Request.Context(), &users.ShutdownRequest{
		Action: users.ShutdownRequest_Reboot,
	})
}
