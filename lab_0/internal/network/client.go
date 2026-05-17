package network

import "github.com/gorilla/websocket"

type Client struct {
	Conn *websocket.Conn
	Hub  *Hub
}
