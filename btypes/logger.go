package btypes

import (
	"io"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// LogTarget 代表Log的目标地
type LogTarget uint

const (
	LogStderr LogTarget = 1 << iota
	LogFile
)

type Logger struct {
	*zap.SugaredLogger
}

func getLoggerLevel(lvl string) zapcore.Level {
	switch lvl {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "dpanic":
		return zapcore.DPanicLevel
	case "panic":
		return zapcore.PanicLevel
	case "fatal":
		return zapcore.FatalLevel
	}

	return zapcore.InfoLevel
}

func NewLogger(filename, lvl string, logTargets LogTarget) Logger {
	level := getLoggerLevel(lvl)

	var writers []io.Writer
	if logTargets&LogStderr == LogStderr {
		writers = append(writers, os.Stderr)
	}
	if logTargets&LogFile == LogFile {
		if filename == "" {
			panic("未设置文件名，但要求文件记录Log")
		}
		writers = append(writers, &lumberjack.Logger{
			Filename:  filename,
			MaxAge:    30,
			MaxSize:   10,
			LocalTime: true,
		})
	}

	syncWriter := zapcore.AddSync(io.MultiWriter(writers...))
	encoder := zap.NewProductionEncoderConfig()
	encoder.EncodeTime = zapcore.ISO8601TimeEncoder
	core := zapcore.NewCore(zapcore.NewJSONEncoder(encoder), syncWriter, zap.NewAtomicLevelAt(level))
	log := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	return Logger{log.Sugar()}
}

func (logger *Logger) Debug(args ...interface{}) {
	logger.SugaredLogger.Debug(args...)
}

func (logger *Logger) Debugf(template string, args ...interface{}) {
	logger.SugaredLogger.Debugf(template, args...)
}

func (logger *Logger) Info(args ...interface{}) {
	logger.SugaredLogger.Info(args...)
}

func (logger *Logger) Infof(template string, args ...interface{}) {
	logger.SugaredLogger.Infof(template, args...)
}

func (logger *Logger) Warn(args ...interface{}) {
	logger.SugaredLogger.Warn(args...)
}

func (logger *Logger) Warnf(template string, args ...interface{}) {
	logger.SugaredLogger.Warnf(template, args...)
}

func (logger *Logger) Error(args ...interface{}) {
	logger.SugaredLogger.Error(args...)
}

func (logger *Logger) Errorf(template string, args ...interface{}) {
	logger.SugaredLogger.Errorf(template, args...)
}

func (logger *Logger) DPanic(args ...interface{}) {
	logger.SugaredLogger.DPanic(args...)
}

func (logger *Logger) DPanicf(template string, args ...interface{}) {
	logger.SugaredLogger.DPanicf(template, args...)
}

func (logger *Logger) Panic(args ...interface{}) {
	logger.SugaredLogger.Panic(args...)
}

func (logger *Logger) Panicf(template string, args ...interface{}) {
	logger.SugaredLogger.Panicf(template, args...)
}

func (logger *Logger) Fatal(args ...interface{}) {
	logger.SugaredLogger.Fatal(args...)
}

func (logger *Logger) Fatalf(template string, args ...interface{}) {
	logger.SugaredLogger.Fatalf(template, args...)
}
