package btypes

import (
	"os"
	"path"
	"strings"

	"github.com/eruca/bisel/utils"
)

// *********************** Logging ******************************
type Logging struct {
	Filename    string
	Level       string
	StderrColor bool
	StderrLevel string
}

// setLevel 因为存在输入字符串不合格
// 默认只接受一下情况，并正规化
func setLevel(pStr *string) {
	switch strings.ToLower(*pStr) {
	case "", "info":
		*pStr = "info"
	case "debug":
		*pStr = "debug"
	case "warn":
		*pStr = "warn"
	case "error":
		*pStr = "error"
	default:
		panic("unknown level:" + *pStr)
	}
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

	setLevel(&log.Level)
	setLevel(&log.StderrLevel)
}

// *******************************************************************
// JWTConfig
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

// ********************************* AppConfig *********************
type AppConfig struct {
	DatabaseHost string
	Addr         string
	QuerySize    int
}

func (app *AppConfig) SetDefault() {
	if app.DatabaseHost == "" {
		app.DatabaseHost = "localhost"
	}

	if app.Addr == "" {
		app.Addr = "9000"
	}

	if app.QuerySize == 0 {
		app.QuerySize = DEFAULT_QUERY_SIZE
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
