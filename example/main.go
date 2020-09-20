package main

import (
	"log"
	"net/http"
	"time"

	"github.com/danielthank/websocket-pubsub"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
)

var (
	redisClient *redis.Client
	upgrader    = websocket.Upgrader{}
	config      = &wspubsub.Config{
		PingPeriod:     50 * time.Second,
		PongWait:       60 * time.Second,
		WriteWait:      10 * time.Second,
		MaxMessageSize: 8192,
	}
)

func serveWs(w http.ResponseWriter, r *http.Request) {
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	wsPubSub := &wspubsub.WSPubSub{
		WSConn:      wsConn,
		RedisClient: redisClient,
		Config:      config,
	}

	go wsPubSub.Run()
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "home.html")
}

func main() {
	redisClient = redis.NewClient(&redis.Options{
		Addr: ":6379",
	})

	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", serveWs)
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", nil))
}
