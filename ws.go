package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var hub = NotificationHub{}

func init() {
	hub.client2channel = make(map[*websocket.Conn]string)
	hub.clients = make(map[string][]*websocket.Conn)
	hub.msgC = make(chan Msg, 100)
	hub.closeC = make(chan struct{})
}

type NotificationHub struct {
	clients map[string][]*websocket.Conn

	client2channel map[*websocket.Conn]string
	lock           sync.RWMutex
	msgC           chan Msg
	closeC         chan struct{}
}

type Msg struct {
	Content string
	Channel string
}

func (h *NotificationHub) Join(channel string, conn *websocket.Conn) {
	h.lock.Lock()
	h.client2channel[conn] = channel
	h.clients[channel] = append(h.clients[channel], conn)
	h.lock.Unlock()
}

func (h *NotificationHub) Leave(conn *websocket.Conn) {
	h.lock.Lock()
	defer h.lock.Unlock()
	channel, exist := h.client2channel[conn]
	if !exist {
		return
	}
	delete(h.client2channel, conn)
	clients := []*websocket.Conn{}

	for _, c := range h.clients[channel] {
		if c != conn {
			clients = append(clients, c)
		}
	}

	h.clients[channel] = clients
}

func (h *NotificationHub) broadcast(channel string, msg []byte) {
	h.lock.RLock()
	closed := []*websocket.Conn{}
	clients := h.clients[channel]
	if len(clients) == 0 {
		h.lock.RUnlock()
		return
	}
	for _, c := range clients {
		c.SetWriteDeadline(time.Now().Add(time.Second))
		writer, err := c.NextWriter(websocket.TextMessage)
		if err != nil {
			fmt.Println("failed to write", err)
			closed = append(closed, c)
		} else {
			writer.Write(msg)
		}
	}
	h.lock.RUnlock()
	if len(closed) > 0 {
		for _, c := range closed {
			h.Leave(c)
			c.Close()
		}
	}
}

func (h *NotificationHub) Close() {
	close(h.closeC)
}

func (h *NotificationHub) Start() {
	for {
		select {
		case msg := <-h.msgC:
			h.broadcast(msg.Channel, []byte(msg.Content))
		case <-h.closeC:
			return
		}
	}
}
