package config

import (
	"log"
	"os"

	qstash "github.com/upstash/qstash-go"
)

var QStashClient *qstash.Client
var QStashReceiver *qstash.Receiver

func InitQStash() {
	token := os.Getenv("QSTASH_TOKEN")
	currentKey := os.Getenv("QSTASH_CURRENT_SIGNING_KEY")
	nextKey := os.Getenv("QSTASH_NEXT_SIGNING_KEY")
	baseURL := os.Getenv("QSTASH_URL") // empty = use production default

	if token == "" {
		log.Println("[QStash] QSTASH_TOKEN not set — QStash scheduling disabled (dev: using in-process timer)")
		return
	}

	opts := qstash.Options{Token: token}
	if baseURL != "" {
		opts.Url = baseURL
		log.Printf("[QStash] Using custom base URL: %s", baseURL)
	}
	QStashClient = qstash.NewClientWith(opts)

	if currentKey != "" && nextKey != "" {
		QStashReceiver = qstash.NewReceiver(currentKey, nextKey)
		log.Println("[QStash] Client and Receiver initialized ✓")
	} else {
		log.Println("[QStash] WARNING: Signing keys not set — webhook signature verification disabled")
	}
}
