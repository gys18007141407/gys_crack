package logger

import (
	"crack_front/src/config"
	"github.com/sirupsen/logrus"
	"os"
	"path"
)

type BaseLogger struct {
	baseLogger 					*logrus.Logger
	fp 							*os.File
}

// 写日志
func (This *BaseLogger) TraceLog (args ...interface{})  {
	This.baseLogger.Traceln(args...)
}
func (This *BaseLogger) DebugLog (args ...interface{})  {
	This.baseLogger.Debugln(args...)
}
func (This *BaseLogger) InfoLog (args ...interface{})  {
	This.baseLogger.Infoln(args...)
}
func (This *BaseLogger) WarnLog (args ...interface{})  {
	This.baseLogger.Warn(args...)
}
func (This *BaseLogger) FatalLog (args ...interface{})  {
	This.baseLogger.Fatalln(args...)
}
func (This *BaseLogger) PanicLog (args ...interface{})  {
	This.baseLogger.Panic(args...)
}

var (
	Logger				*BaseLogger
)


func AliasToLogLevel(level string) logrus.Level{
	switch level {
	case "trace":
		return logrus.TraceLevel
	case "debug":
		return logrus.DebugLevel
	case "info":
		return logrus.InfoLevel
	case "warn":
		return logrus.WarnLevel
	case "fatal":
		return logrus.FatalLevel
	case "panic":
		return logrus.PanicLevel
	default:
		return logrus.TraceLevel
	}
}

func InitLogger() (err error) {
	var(
		baseLogger 					*logrus.Logger
		fp 							*os.File
	)
	if Logger == nil{
		baseLogger = logrus.New()
		if err = os.MkdirAll(config.Cfg.LogFilePath, 0755); err != nil{
			return err
		}
		if fp, err = os.OpenFile(path.Join(config.Cfg.LogFilePath, config.Cfg.LogFileName), os.O_CREATE | os.O_RDWR | os.O_APPEND, 0644); err != nil{
			return err
		}
		baseLogger.SetOutput(fp)
		baseLogger.SetLevel(AliasToLogLevel(config.Cfg.LogLevel))
		baseLogger.SetFormatter(&logrus.TextFormatter{
			DisableColors:             false,
			DisableQuote:              false,
			DisableTimestamp:          false,
			TimestampFormat:           "[2006_Jan_02 15:04:05]",
			PadLevelText:              false,
		})

		// 赋值单例
		Logger = &BaseLogger{
			baseLogger: baseLogger,
			fp:         fp,
		}
	}
	return
}
