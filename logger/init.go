package logger

import "sync"

var (
	globalLogger Interface = New()
	globalMux    sync.Mutex
)

func SetLogger(l Interface) {
	globalMux.Lock()
	globalLogger = l
	globalMux.Unlock()
}

func Errorf(format string, v ...interface{}) {
	if globalLogger == nil {
		return
	}

	globalLogger.Errorf(format, v...)
}

func Warnf(format string, v ...interface{}) {
	if globalLogger == nil {
		return
	}

	globalLogger.Warnf(format, v...)
}

func Infof(format string, v ...interface{}) {
	if globalLogger == nil {
		return
	}

	globalLogger.Infof(format, v...)
}

func Debugf(format string, v ...interface{}) {
	if globalLogger == nil {
		return
	}

	globalLogger.Debugf(format, v...)
}
