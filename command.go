package wspubsub

import (
	"bytes"
	"errors"
	"strings"
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

type commandType int

const (
	publish commandType = iota
	subscribe
	psubscribe
	unsubscribe
	punsubscribe
)

type command struct {
	kind     commandType
	channels []string
	payload  string
}

func parseCommand(message []byte) (*command, error) {
	message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
	str := string(message)
	strs := strings.SplitN(str, " ", 2)
	if len(strs) != 2 {
		return nil, errors.New("invalid command")
	}
	switch strings.ToLower(strs[0]) {
	case "publish":
		args := strings.SplitN(strs[1], " ", 2)
		if len(args) < 2 {
			return nil, errors.New("invalid command")
		}
		return &command{
			kind:     publish,
			channels: []string{args[0]},
			payload:  args[1],
		}, nil
	case "subscribe":
		channels := strings.Split(strs[1], " ")
		if len(channels) == 0 {
			return nil, errors.New("invalid command")
		}
		return &command{
			kind:     subscribe,
			channels: channels,
		}, nil
	case "psubscribe":
		channels := strings.Split(strs[1], " ")
		if len(channels) == 0 {
			return nil, errors.New("invalid command")
		}
		return &command{
			kind:     psubscribe,
			channels: channels,
		}, nil
	case "unsubscribe":
		channels := strings.Split(strs[1], " ")
		if len(channels) == 0 {
			return nil, errors.New("invalid command")
		}
		return &command{
			kind:     unsubscribe,
			channels: channels,
		}, nil
	case "punsubscribe":
		channels := strings.Split(strs[1], " ")
		if len(channels) == 0 {
			return nil, errors.New("invalid command")
		}
		return &command{
			kind:     punsubscribe,
			channels: channels,
		}, nil
	default:
		return nil, errors.New("invalid command")
	}
}
