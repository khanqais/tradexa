package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func StreamBid(c *gin.Context) {
	listingIDStr := c.Param("id")
	listingID, err := strconv.ParseUint(listingIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid listing id",
		})
		return
	}
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	clientChan := make(chan []byte, 10)
	StreamHub.AddClient(uint(listingID), clientChan)

	defer StreamHub.RemoveClient(uint(listingID), clientChan)
	c.Writer.Write([]byte(":connected\n\n"))

	c.Writer.Flush()

	for {
		select {
		case msg := <-clientChan:
			fmt.Fprintf(c.Writer, "data: %s\n\n", msg)
			c.Writer.Flush()
		case <-c.Request.Context().Done():
			return
		}
	}
}
