package handlers

import (
	"context"
	"log"
	"net/http"
	"sync"

	"cat-led/internal/pkg/launchericon"
	"cat-led/internal/pkg/zlog"

	gohelper "gitee.com/linakesi/lzc-sdk/lang/go"
	users "gitee.com/linakesi/lzc-sdk/lang/go/common"
	"github.com/gin-gonic/gin"
)

var (
	ledStatus bool // false=off, true=on
	ledMutex  sync.Mutex
)

// InitLedStatus initializes the LED status from the device at startup.
func InitLedStatus(ctx context.Context, logger *zlog.Logger) {
	ledMutex.Lock()
	defer ledMutex.Unlock()
	gw, err := gohelper.NewAPIGateway(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Error creating API gateway for LED status initialization")
		return
	}
	defer gw.Close()

	boxInfo, err := gw.Box.QueryInfo(ctx, nil)
	if err != nil {
		logger.Error().Err(err).Msg("Error querying box info for LED status initialization")
		return
	}

	ledStatus = boxInfo.PowerLed
	if err := launchericon.Update(ledStatus); err != nil {
		log.Printf("Update launcher icon: %v", err)
	}

	log.Printf("LED status initialized: %v", ledStatus)
}

// LedControl toggles the LED state and returns the new status.
func LedControl(c *gin.Context) {
	ledMutex.Lock()
	defer ledMutex.Unlock()
	ctx := c.Request.Context()

	gw, err := gohelper.NewAPIGateway(ctx)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer gw.Close()

	boxInfo, err := gw.Box.QueryInfo(ctx, nil)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	newStatus := !boxInfo.PowerLed

	if boxInfo.PowerLed {
		log.Println("led is on, turning off")
	} else {
		log.Println("led is off, turning on")
	}

	_, err = gw.Box.ChangePowerLed(ctx, &users.ChangePowerLedRequest{
		PowerLed: newStatus,
	})
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	ledStatus = newStatus
	if err := launchericon.Update(ledStatus); err != nil {
		log.Printf("Update launcher icon: %v", err)
	}
	c.JSON(http.StatusOK, gin.H{"status": ledStatus})
}

// GetLedStatus returns the current LED status from the device.
func GetLedStatus(c *gin.Context) {
	ledMutex.Lock()
	defer ledMutex.Unlock()
	ctx := c.Request.Context()

	gw, err := gohelper.NewAPIGateway(ctx)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer gw.Close()

	boxInfo, err := gw.Box.QueryInfo(ctx, nil)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	ledStatus = boxInfo.PowerLed
	if err := launchericon.Update(ledStatus); err != nil {
		log.Printf("Update launcher icon: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{"status": ledStatus})
}

// SetLedStatus serializes scheduled changes with manual controls and status
// reads so that an older observation cannot overwrite a newer launcher icon.
func SetLedStatus(ctx context.Context, status bool) error {
	ledMutex.Lock()
	defer ledMutex.Unlock()
	gw, err := gohelper.NewAPIGateway(ctx)
	if err != nil {
		return err
	}
	defer gw.Close()
	if _, err := gw.Box.ChangePowerLed(ctx, &users.ChangePowerLedRequest{PowerLed: status}); err != nil {
		return err
	}
	ledStatus = status
	if err := launchericon.Update(status); err != nil {
		log.Printf("Update launcher icon: %v", err)
	}
	return nil
}
