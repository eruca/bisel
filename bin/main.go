package main

import (
	"log"
	"time"

	"github.com/eruca/bisel/bin/models/journal"
	"github.com/eruca/bisel/bin/models/users"
	"github.com/eruca/bisel/btypes"
	"github.com/eruca/bisel/manager"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	logging, writer := btypes.NewLogger("hello.log", "info", btypes.LogStderr|btypes.LogFile)
	newLogger := logger.New(
		log.New(writer, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold: time.Second,   // Slow SQL threshold
			LogLevel:      logger.Silent, // Log level
			Colorful:      true,
		},
	)

	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{Logger: newLogger})
	if err != nil {
		panic("failed to connect database")
	}

	// db 作为元数据存储的数据库
	// dsn := "host=localhost user=nick password=nickwill dbname=icu sslmode=disable TimeZone=Asia/Shanghai"
	// db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	// if err != nil {
	// 	log.Println("cannot open db", "err", err)
	// 	os.Exit(1)
	// }
	// 开启debug
	db = db.Debug()

	config := manager.LoadConfigFile()

	// Manager
	manager := manager.New(db, btypes.NewCacher(logging), logging, config, nil, &journal.Journal{}, &users.User{})
	// 配置gin
	engine := gin.Default()
	engine.Use(cors())
	manager.InitSystem(engine, nil)

	logging.Fatal("Router.Run:", "err", engine.Run(":"+(config.App.Addr)))
}

// CORSMiddleware 实现跨域
func cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
