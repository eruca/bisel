package btypes

import (
	"os"
	"path"

	"github.com/eruca/bisel/utils"
)

type Logging struct {
	Filename string
	Level    string
}

func (log *Logging) SetDefault() {
	if log.Filename == "" {
		log.Filename = "./logs/zap.log"
	}

	dir := path.Dir(log.Filename)
	if utils.IsNotExist(dir) {
		os.MkdirAll(dir, os.ModePerm)
	}

	if log.Level == "" {
		log.Level = "info"
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
