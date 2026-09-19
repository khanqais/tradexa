package workers

import (
	"fmt"
	"net/http"
	"time"
)

const healthInterval = 30 * time.Second

var pingTargets = []string{
	"https://tradexa-hdhq.onrender.com/api/health",
	"https://tradexa-hdhq.onrender.com/api/keep-alive",
}

func StartHealthPing() {
	client := &http.Client{Timeout: 10 * time.Second}
	ticker := time.NewTicker(healthInterval)
	defer ticker.Stop()

	fmt.Printf("[HealthPing] Started — pinging %d endpoint(s) every %s\n", len(pingTargets), healthInterval)

	for range ticker.C {
		for _, url := range pingTargets {
			resp, err := client.Get(url)
			if err != nil {
				fmt.Printf("[HealthPing] ERROR %s: %v\n", url, err)
				continue
			}
			resp.Body.Close()
			fmt.Printf("[HealthPing] %s → %d\n", url, resp.StatusCode)
		}
	}
}
