package handlers

import (
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

var Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		// Allow localhost and 127.0.0.1 in development
		if origin == "" { // Direct connection without Origin header
			return true
		}
		return strings.Contains(origin, "localhost") || strings.Contains(origin, "127.0.0.1")
	},
}
