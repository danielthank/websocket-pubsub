package wspubsub

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
)

type RedisMessage struct {
	Channel      string   `json:"channel"`
	Pattern      string   `json:"pattern"`
	Payload      string   `json:"payload"`
	PayloadSlice []string `json:"-"`
}

type Config struct {
	PingPeriod     time.Duration
	PongWait       time.Duration
	WriteWait      time.Duration
	MaxMessageSize int64
}

type WSPubSub struct {
	WSConn      *websocket.Conn
	RedisClient *redis.Client
	Config      *Config
	pubsub      *redis.PubSub
	stop        chan struct{}
	toStop      chan struct{}
	wg          sync.WaitGroup
}

func (t *WSPubSub) Run() {
	defer func() {
		t.pubsub.Close()
		t.WSConn.Close()
	}()

	// init an empty pubsub
	t.pubsub = t.RedisClient.Subscribe(context.Background())
	err := t.pubsub.Ping(context.Background())
	if err != nil {
		t.WSConn.Close()
		return
	}

	t.stop = make(chan struct{})
	t.toStop = make(chan struct{}, 1)
	t.wg = sync.WaitGroup{}
	t.wg.Add(3)

	// Moderator
	go func() {
		defer t.wg.Done()
		<-t.toStop
		close(t.stop)
	}()
	go t.readPump()
	go t.writePump()

	t.wg.Wait()
}

func (t *WSPubSub) readPump() {
	defer t.wg.Done()
	t.WSConn.SetReadLimit(t.Config.MaxMessageSize)
	t.WSConn.SetReadDeadline(time.Now().Add(t.Config.PongWait))
	t.WSConn.SetPongHandler(
		func(string) error {
			t.WSConn.SetReadDeadline(time.Now().Add(t.Config.PongWait))
			return nil
		},
	)

	tryStop := func() {
		select {
		case t.toStop <- struct{}{}:
		default:
		}
	}

	for {
		// try-receive. Exit as early as possible
		select {
		case <-t.stop:
			return
		default:
		}
		_, message, err := t.WSConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Println(err)
			}
			tryStop()
			return
		}
		command, err := parseCommand(message)
		if err != nil {
			log.Println(err)
			continue
		}
		err = t.exec(command)
		if err != nil {
			log.Println(err)
			continue
		}
	}
}

func (t *WSPubSub) writePump() {
	ticker := time.NewTicker(t.Config.PingPeriod)
	defer func() {
		ticker.Stop()
		t.wg.Done()
	}()

	tryStop := func() {
		select {
		case t.toStop <- struct{}{}:
		default:
		}
	}

	for {
		// try-receive. Exit as early as possible
		select {
		case <-t.stop:
			return
		default:
		}

		select {
		case <-t.stop:
			return
		case message, ok := <-t.pubsub.Channel():
			if !ok {
				tryStop()
				return
			}
			t.WSConn.SetWriteDeadline(time.Now().Add(t.Config.WriteWait))

			w, err := t.WSConn.NextWriter(websocket.TextMessage)
			if err != nil {
				tryStop()
				return
			}
			if messageBytes, err := json.Marshal(RedisMessage(*message)); err == nil {
				w.Write(messageBytes)
			}

			if err := w.Close(); err != nil {
				tryStop()
				return
			}
		case <-ticker.C:
			t.WSConn.SetWriteDeadline(time.Now().Add(t.Config.WriteWait))
			if err := t.WSConn.WriteMessage(websocket.PingMessage, nil); err != nil {
				tryStop()
				return
			}
		}
	}
}

func (t *WSPubSub) exec(cmd *command) error {
	ctx := context.Background()
	switch cmd.kind {
	case publish:
		if err := t.RedisClient.Publish(ctx, cmd.channels[0], cmd.payload).Err(); err != nil {
			return err
		}
	case subscribe:
		if err := t.pubsub.Subscribe(ctx, cmd.channels...); err != nil {
			return err
		}
	case psubscribe:
		if err := t.pubsub.PSubscribe(ctx, cmd.channels...); err != nil {
			return err
		}
	case unsubscribe:
		if err := t.pubsub.Unsubscribe(ctx, cmd.channels...); err != nil {
			return err
		}
	case punsubscribe:
		if err := t.pubsub.PUnsubscribe(ctx, cmd.channels...); err != nil {
			return err
		}
	}
	return nil
}
