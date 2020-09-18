package logger

import (
	"io"
	"os"

	"github.com/golib/zerolog"
)

var (
	zlog = zerolog.New(os.Stderr).With().Timestamp().Logger()
)

type Logger struct {
	log zerolog.Logger
}

func New(w ...io.Writer) *Logger {
	nlog := zlog.Hook(zerolog.HookFunc(func(e *zerolog.Event, _ zerolog.Level, _ string) {
		e.Str("sdk", "discovery")
	})).With().Logger()
	if len(w) > 0 {
		nlog = nlog.Output(w[0])
	}

	return &Logger{
		log: nlog,
	}
}

func (l *Logger) Errorf(format string, v ...interface{}) {
	l.log.Error().Msgf(format, v...)
}

func (l *Logger) Warnf(format string, v ...interface{}) {
	l.log.Warn().Msgf(format, v...)
}

func (l *Logger) Infof(format string, v ...interface{}) {
	l.log.Info().Msgf(format, v...)
}

func (l *Logger) Debugf(format string, v ...interface{}) {
	l.log.Debug().Msgf(format, v...)
}
