package logger

import (
	"fmt"
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

type LogLevel int

const (
	LogDebug LogLevel = iota
	LogInfo
	LogWarn
	LogError
)

// func convertToLogLevel(lvl string) LogLevel {
// 	switch lvl {
// 	case "debug":
// 		return LogDebug
// 	case "info":
// 		return LogInfo
// 	case "warn":
// 		return LogWarn
// 	case "error":
// 		return LogError
// 	}
// 	panic("should not happened:'" + lvl + "'")
// }

type Logger interface {
	Debugf(string, ...interface{})
	Infof(string, ...interface{})
	Warnf(string, ...interface{})
	Errorf(string, ...interface{})
}

// Colors
const (
	Reset       = "\033[0m"
	Red         = "\033[31m"
	Green       = "\033[32m"
	Yellow      = "\033[33m"
	Blue        = "\033[34m"
	Magenta     = "\033[35m"
	Cyan        = "\033[36m"
	White       = "\033[37m"
	BlueBold    = "\033[34;1m"
	MagentaBold = "\033[35;1m"
	RedBold     = "\033[31;1m"
	YellowBold  = "\033[33;1m"

	debugStr      = "[debug] %s\n"
	debugStrColor = Blue + "[debug]" + Reset + Cyan + "%s\n" + Reset
	infoStr       = "[info] %s\n"
	infoStrColor  = Green + "[info] " + Reset + Green + "%s\n" + Reset
	warnStr       = "[warn] %s\n"
	warnStrColor  = Magenta + "[warn] " + Reset + BlueBold + "%s\n" + Reset
	errStr        = "[error] %s\n"
	errStrColor   = Red + "[error] " + Reset + Magenta + "%s\n" + Reset
)

func NewLogger(logTarget LogTarget) (targets MultiTargets) {
	// if logTarget&LogStderr == LogStderr {
	targets = append(targets, &stderr{
		target:   target{os.Stderr},
		level:    LogDebug,
		colorful: true,
	})
	// }

	// if logTarget&LogFile == LogFile {
	// 	targets = append(targets, NewFileLogger(logConfig.Filename, convertToLogLevel(logConfig.Level)))
	// }

	return
}

// ********************************** MultiTargets *********************************************
type MultiTargets []Logger

func (mt MultiTargets) Debugf(tmpl string, args ...interface{}) {
	for _, target := range mt {
		target.Debugf(tmpl, args...)
	}
}

func (mt MultiTargets) Infof(tmpl string, args ...interface{}) {
	for _, target := range mt {
		target.Infof(tmpl, args...)
	}
}

func (mt MultiTargets) Warnf(tmpl string, args ...interface{}) {
	for _, target := range mt {
		target.Warnf(tmpl, args...)
	}
}

func (mt MultiTargets) Errorf(tmpl string, args ...interface{}) {
	for _, target := range mt {
		target.Errorf(tmpl, args...)
	}
}

type target struct {
	io.Writer
}

func (t target) Debugf(tmpl string, args ...interface{}) {
	t.Write([]byte(fmt.Sprintf(tmpl, args...)))
}

func (t target) Infof(tmpl string, args ...interface{}) {
	t.Write([]byte(fmt.Sprintf(tmpl, args...)))

}

func (t target) Warnf(tmpl string, args ...interface{}) {
	t.Write([]byte(fmt.Sprintf(tmpl, args...)))
}

func (t target) Errorf(tmpl string, args ...interface{}) {
	t.Write([]byte(fmt.Sprintf(tmpl, args...)))
}

// ***************************** stderr logger ********************************
type stderr struct {
	target
	level    LogLevel
	colorful bool
}

func (sd *stderr) Debugf(tmpl string, args ...interface{}) {
	if sd.level <= LogDebug {
		text := fmt.Sprintf(tmpl, args...)

		levelTmpl := debugStr
		if sd.colorful {
			levelTmpl = debugStrColor
		}
		sd.target.Debugf(levelTmpl, text)
	}
}

func (sd *stderr) Infof(tmpl string, args ...interface{}) {
	if sd.level <= LogInfo {
		text := fmt.Sprintf(tmpl, args...)
		levelTmpl := infoStr
		if sd.colorful {
			levelTmpl = infoStrColor
		}
		sd.target.Debugf(levelTmpl, text)
	}
}

func (sd *stderr) Warnf(tmpl string, args ...interface{}) {
	if sd.level <= LogWarn {
		text := fmt.Sprintf(tmpl, args...)
		levelTmpl := warnStr
		if sd.colorful {
			levelTmpl = warnStrColor
		}
		sd.target.Debugf(levelTmpl, text)
	}
}

func (sd *stderr) Errorf(tmpl string, args ...interface{}) {
	if sd.level <= LogError {
		text := fmt.Sprintf(tmpl, args...)
		levelTmpl := errStr
		if sd.colorful {
			levelTmpl = errStrColor
		}
		sd.target.Debugf(levelTmpl, text)
	}
}

//*********************** file logger *******************************
type ZapLogger struct {
	*zap.SugaredLogger
}

func getLogLevel(l LogLevel) zapcore.Level {
	switch l {
	case LogDebug:
		return zapcore.DebugLevel
	case LogInfo:
		return zapcore.InfoLevel
	case LogWarn:
		return zapcore.WarnLevel
	case LogError:
		return zapcore.ErrorLevel
	}
	return zapcore.InfoLevel
}

func NewFileLogger(filename string, lvl LogLevel) Logger {
	if filename == "" {
		panic("未设置文件名，但要求文件记录Log")
	}
	syncWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:  filename,
		MaxAge:    30,
		MaxSize:   10,
		LocalTime: true,
	})
	encoder := zap.NewProductionEncoderConfig()
	encoder.EncodeTime = zapcore.ISO8601TimeEncoder

	level := getLogLevel(lvl)
	core := zapcore.NewCore(zapcore.NewJSONEncoder(encoder), syncWriter, zap.NewAtomicLevelAt(level))
	log := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	return &ZapLogger{SugaredLogger: log.Sugar()}
}

func (logger *ZapLogger) Debugf(template string, args ...interface{}) {
	logger.SugaredLogger.Debugf(template, args...)
}

func (logger *ZapLogger) Infof(template string, args ...interface{}) {
	logger.SugaredLogger.Infof(template, args...)
}

func (logger *ZapLogger) Warnf(template string, args ...interface{}) {
	logger.SugaredLogger.Warnf(template, args...)
}

func (logger *ZapLogger) Errorf(template string, args ...interface{}) {
	logger.SugaredLogger.Errorf(template, args...)
}
