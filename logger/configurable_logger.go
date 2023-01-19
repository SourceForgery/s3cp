package logger

import "github.com/amoghe/distillog"

type LogLevel int8

type ConfigurableVerboseLogger struct {
	ProxyLog    distillog.Logger
	MinLogLevel LogLevel
}

const (
	Debug   LogLevel = 0
	Info             = 1
	Warning          = 2
	Error            = 3
)

func (log ConfigurableVerboseLogger) Debugf(format string, v ...interface{}) {
	if log.MinLogLevel <= Debug {
		log.ProxyLog.Debugf(format, v)
	}
}
func (log ConfigurableVerboseLogger) Debugln(v ...interface{}) {
	if log.MinLogLevel <= Debug {
		log.ProxyLog.Debugln(v)
	}
}
func (log ConfigurableVerboseLogger) Infof(format string, v ...interface{}) {
	if log.MinLogLevel <= Info {
		log.ProxyLog.Infof(format, v)
	}
}
func (log ConfigurableVerboseLogger) Infoln(v ...interface{}) {
	if log.MinLogLevel <= Info {
		log.ProxyLog.Infoln(v)
	}
}
func (log ConfigurableVerboseLogger) Warningf(format string, v ...interface{}) {
	if log.MinLogLevel <= Warning {
		log.ProxyLog.Warningf(format, v)
	}
}
func (log ConfigurableVerboseLogger) Warningln(v ...interface{}) {
	if log.MinLogLevel <= Warning {
		log.ProxyLog.Warningln(v)
	}
}
func (log ConfigurableVerboseLogger) Errorf(format string, v ...interface{}) {
	if log.MinLogLevel <= Error {
		log.ProxyLog.Errorf(format, v)
	}
}
func (log ConfigurableVerboseLogger) Errorln(v ...interface{}) {
	if log.MinLogLevel <= Error {
		log.ProxyLog.Errorln(v)
	}
}
func (log ConfigurableVerboseLogger) Close() error {
	return log.ProxyLog.Close()
}
