package handlers

import (
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/websocket"
)

var Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")

		if origin == "" {
			return true
		}

		if strings.Contains(origin, "localhost") || strings.Contains(origin, "127.0.0.1") {
			return true
		}

		frontendURL := strings.TrimSpace(os.Getenv("FRONTEND_URL"))
		if frontendURL != "" && strings.EqualFold(origin, frontendURL) {
			return true
		}

		return false
	},
}
