package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/eruca/bisel/manager"
	"github.com/eruca/bisel/models/journal"
	"github.com/eruca/bisel/net/ws"
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
	manager := manager.New(&types.DB{Gorm: db}, types.NewCacher(), nil, &journal.Journal{})

	router := gin.Default()
	router.Use(cors())

	router.POST("/:table/:acid", func(c *gin.Context) {
		table := c.Param("table")
		acid := c.Param("acid")
		router := fmt.Sprintf("%s/%s", table, acid)
		req := types.FromHttpRequest(router, c.Request.Body)
		log.Printf("http request from client: %-v\n", req)

		manager.TakeAction(c.Writer, req, c.Request)
	})

	// 构建读入信息后的处理函数
	processMixHttpRequest := func(httpReq *http.Request) ws.Process {
		return func(send chan<- []byte, msg []byte) {
			req := types.NewRequest(bytes.TrimSpace(msg))
			log.Printf("websocket request from client: %-v\n", req)
			manager.TakeAction(types.NewChanWriter(send), req, httpReq)
		}
	}
	// 连接成功后马上发送的数据
	connected := func(send chan<- []byte) {
		log.Println("Connected now, will send some data to client")
		manager.Connected(send)
	}
	wsHandler := ws.WebsocketHandler(processMixHttpRequest, connected)
	router.GET("/ws", func(c *gin.Context) {
		wsHandler(c.Writer, c.Request)
	})

	log.Fatalln("Router.Run:", "err", router.Run(":"+(*addr)))
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
