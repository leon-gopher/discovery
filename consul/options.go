package consul

import (
	"time"

	"github.com/leon-gopher/discovery/dumper"
)

type option struct {
	stale      bool
	agentCache bool

	// degrade with passingOnly=false settings
	passingOnly bool
	threshold   float32

	// local file cache interface
	dumper            dumper.Dumper
	watchDumpInterval time.Duration

	watchWaitTime        time.Duration
	debug                bool
	firstFetchUseCatalog bool

	calmInterval time.Duration
}

func (o *option) enableDegrade() bool {
	return o.threshold > 0
}

type ConsulOption func(*option)

//第一次请求使用catalog接口
func WithFirstUseCatalog(isCatalog bool) ConsulOption {
	return func(o *option) {
		o.firstFetchUseCatalog = isCatalog
	}
}

func WithStale(stale bool) ConsulOption {
	return func(o *option) {
		o.stale = stale
	}
}

func WithAgentCache(cache bool) ConsulOption {
	return func(o *option) {
		o.agentCache = cache
	}
}

func WithPassingOnly(passingOnly bool) ConsulOption {
	return func(o *option) {
		o.passingOnly = passingOnly
	}
}

func WithDegrade(threshold float32) ConsulOption {
	return func(o *option) {
		o.threshold = threshold
	}
}

func WithDumper(dumper dumper.Dumper) ConsulOption {
	return func(o *option) {
		o.dumper = dumper
	}
}

func WithDumpInterval(interval time.Duration) ConsulOption {
	return func(o *option) {
		o.watchDumpInterval = interval
	}
}

// WithWatchWaitTime wait >= 30s and wait <= 10m
func WithWatchWaitTime(wait time.Duration) ConsulOption {
	return func(o *option) {
		if wait.Seconds() < 30 {
			wait = 30 * time.Second
		}

		if wait.Minutes() > 10 {
			wait = 10 * time.Minute
		}

		o.watchWaitTime = wait
	}
}

func WithDebug(debug bool) ConsulOption {
	return func(o *option) {
		o.debug = debug
	}
}

// WithCalmInterval 设置阀值监控时间
func WithCalmInterval(interval time.Duration) ConsulOption {
	return func(o *option) {
		o.calmInterval = interval
	}
}
