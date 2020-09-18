package consul

import (
	"os"
	"strings"
	"time"
)

const (
	DefaultServiceWeight                  = 100
	DefaultServiceCheckInterval           = "5s"
	DefaultServiceDeregisterCriticalAfter = "24h"
	DefaultDegradeThreshold               = 0.8
	DefaultWatchWaitTime                  = 3 * time.Minute
	DefaultWatchDumpInterval              = 3 * time.Hour
	DefaultWatchRollingWindowSize         = 10
	DefaultCalmInterval                   = 1 * time.Hour
	DefaultRetryTimes                     = 3
)

// consul 降级策略
type DegradeStatus int

const (
	WatchNormalize DegradeStatus = iota
	WatchDegraded
)

var DefaultServiceMeta map[string]string

func init() {
	meta := map[string]string{
		"cloud":     "aliyun",
		"container": "vm",
		"registry":  "consul",
	}

	hostname, err := os.Hostname()
	if err == nil && len(hostname) > 0 {
		fields := strings.Split(hostname, "-")
		meta["zone"] = fields[0]
	}

	DefaultServiceMeta = meta
}
