package btypes

import (
	"os"
	"path"
	"strings"

	"github.com/eruca/bisel/utils"
)

type Logging struct {
	Filename    string
	Level       string
	StderrColor bool
	StderrLevel string
}

func (log *Logging) SetDefault() {
	if log.Filename == "" {
		log.Filename = "./logs/zap.log"
	} else if !strings.HasPrefix(log.Filename, "./logs/") && !strings.HasPrefix(log.Filename, "logs/") {
		// 所有的logs都在logs目录下，如果未以logs/开始，就添加
		log.Filename = path.Join("logs/", log.Filename)
	}

	dir := path.Dir(log.Filename)
	if utils.IsNotExist(dir) {
		os.MkdirAll(dir, os.ModePerm)
	}

	switch strings.ToLower(log.Level) {
	case "", "info":
		log.Level = "info"
	case "debug":
		log.Level = "debug"
	case "warn":
		log.Level = "warn"
	case "error":
		log.Level = "error"
	default:
		panic("unknown level:" + log.Level)
	}

	switch strings.ToLower(log.StderrLevel) {
	case "":
		log.StderrLevel = "info"
	case "debug":
		log.StderrLevel = "debug"
	case "warn":
		log.StderrLevel = "warn"
	case "error":
		log.StderrLevel = "error"
	default:
		panic("unknown level:" + log.StderrLevel)
	}
}

type JWTConfig struct {
	Salt   string
	Expire int
}

func (jwt *JWTConfig) SetDefault() {
	if jwt.Salt == "" {
		jwt.Salt = "salt"
	}

	if jwt.Expire == 0 {
		jwt.Expire = 24
	}
}

type AppConfig struct {
	Addr      string
	QuerySize int
}

func (app *AppConfig) SetDefault() {
	if app.Addr == "" {
		app.Addr = "9000"
	}

	if app.QuerySize == 0 {
		app.QuerySize = 20
	}
}

type Config struct {
	Logger Logging
	JWT    JWTConfig
	App    AppConfig
}

func (conf *Config) SetDefault() {
	conf.Logger.SetDefault()
	conf.JWT.SetDefault()
	conf.App.SetDefault()
}
