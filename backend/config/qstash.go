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
	if token == "" {
		log.Println("[QStash] QSTASH_TOKEN not set — QStash disabled (using in-process timer fallback)")
		return
	}

	// The SDK reads QSTASH_URL, QSTASH_CURRENT_SIGNING_KEY, and QSTASH_NEXT_SIGNING_KEY
	// from the environment itself — no need to pass them manually.
	QStashClient = qstash.NewClientWithEnv()

	baseURL := os.Getenv("QSTASH_URL")
	if baseURL == "" {
		log.Println("[QStash] Client initialized → Upstash Cloud (production)")
	} else {
		log.Printf("[QStash] Client initialized → %s", baseURL)
	}

	currentKey := os.Getenv("QSTASH_CURRENT_SIGNING_KEY")
	nextKey := os.Getenv("QSTASH_NEXT_SIGNING_KEY")
	if currentKey != "" && nextKey != "" {
		QStashReceiver = qstash.NewReceiver(currentKey, nextKey)
		log.Println("[QStash] Receiver initialized — webhook signature verification enabled ✓")
	} else {
		log.Println("[QStash] WARNING: Signing keys not set — webhook signature verification disabled")
	}
}
