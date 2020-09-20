module github.com/danielthank/websocket-pubsub/example

go 1.15

replace github.com/danielthank/websocket-pubsub => ../

require (
	github.com/danielthank/websocket-pubsub v0.0.0-00010101000000-000000000000
	github.com/go-redis/redis/v8 v8.1.3
	github.com/gorilla/websocket v1.4.2
)
