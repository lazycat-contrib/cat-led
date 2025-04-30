package handlers

import (
	"context"
	"log"
	"sync"

	gohelper "gitee.com/linakesi/lzc-sdk/lang/go"
	users "gitee.com/linakesi/lzc-sdk/lang/go/common"
	"github.com/gin-gonic/gin"
)

// 使用全局变量追踪LED状态
var (
	ledStatus bool // false表示关闭，true表示开启
	ledMutex  sync.Mutex
)

// InitLedStatus 在程序启动时初始化LED状态
func InitLedStatus(ctx context.Context) {
	gw, err := gohelper.NewAPIGateway(ctx)
	if err != nil {
		log.Printf("Error creating API gateway for LED status initialization: %v", err)
		return
	}
	defer gw.Close()

	boxInfo, err := gw.Box.QueryInfo(ctx, nil)
	if err != nil {
		log.Printf("Error querying box info for LED status initialization: %v", err)
		return
	}

	ledMutex.Lock()
	ledStatus = boxInfo.PowerLed
	ledMutex.Unlock()

	log.Printf("LED status initialized: %v", ledStatus)
}

// LedControl 控制LED开关
func LedControl(c *gin.Context) {
	gw, err := gohelper.NewAPIGateway(c.Request.Context())
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	defer gw.Close()
	boxInfo, err := gw.Box.QueryInfo(c.Request.Context(), nil)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	ledMutex.Lock()
	defer ledMutex.Unlock()

	// 更新全局状态变量
	if boxInfo.PowerLed {
		log.Println("led is on, turning off")
		gw.Box.ChangePowerLed(c.Request.Context(), &users.ChangePowerLedRequest{
			PowerLed: false,
		})
		ledStatus = false
	} else {
		log.Println("led is off, turning on")
		gw.Box.ChangePowerLed(c.Request.Context(), &users.ChangePowerLedRequest{
			PowerLed: true,
		})
		ledStatus = true
	}

	c.JSON(200, gin.H{"status": ledStatus})
}

// GetLedStatus 获取当前LED状态
func GetLedStatus(c *gin.Context) {
	gw, err := gohelper.NewAPIGateway(c.Request.Context())
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	defer gw.Close()
	boxInfo, err := gw.Box.QueryInfo(c.Request.Context(), nil)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	ledMutex.Lock()
	ledStatus = boxInfo.PowerLed
	ledMutex.Unlock()

	c.JSON(200, gin.H{"status": ledStatus})
}
