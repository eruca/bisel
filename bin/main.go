package main

import (
	"flag"
	"log"

	"github.com/eruca/bisel/manager"
	"github.com/eruca/bisel/models/journal"
	"github.com/eruca/bisel/types"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	addr = flag.String("addr", "9000", "the port of the server")
)

func main() {
	flag.Parse()
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
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

	// Manager
	manager := manager.New(db, types.NewCacher(), nil, &journal.Journal{})
	// 配置gin
	engine := gin.Default()
	engine.Use(cors())
	manager.InitSystem(engine)

	log.Fatalln("Router.Run:", "err", engine.Run(":"+(*addr)))
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
