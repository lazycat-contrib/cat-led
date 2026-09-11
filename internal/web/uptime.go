package web

import (
	"fmt"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// readSystemUptime reads Linux's boot-relative clock, including time suspended.
// It is independent of when this application process started.
func readSystemUptime(path string) (int64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return 0, fmt.Errorf("empty system uptime")
	}
	seconds, err := strconv.ParseFloat(fields[0], 64)
	if err != nil || math.IsNaN(seconds) || seconds < 0 || seconds >= math.MaxInt64 {
		return 0, fmt.Errorf("invalid system uptime")
	}
	return int64(seconds), nil
}

func handleSystemUptime(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	seconds, err := readSystemUptime("/proc/uptime")
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "System uptime unavailable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"uptime_seconds": seconds})
}
