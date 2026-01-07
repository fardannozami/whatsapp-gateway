package http

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
)

func (h *Handler) SessionsStream(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()

	ctx := c.Request.Context()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	lastPayload := ""
	lastSent := time.Time{}

	send := func() {
		items, err := h.sessUC.Execute(ctx)
		if err != nil {
			payload := SessionsStreamResponse{
				Status: "failed",
				Detail: err.Error(),
			}
			c.SSEvent("sessions", payload)
			c.Writer.Flush()
			lastSent = time.Now()
			return
		}

		sessions := make([]SessionItemResponse, 0, len(items))
		for _, item := range items {
			sessions = append(sessions, SessionItemResponse{
				Session:  item.Session,
				ID:       item.ID,
				PushName: item.PushName,
				Status:   item.Status,
			})
		}

		payload := SessionsStreamResponse{
			Status:   "ok",
			Sessions: sessions,
		}

		data, err := json.Marshal(payload)
		if err != nil {
			return
		}

		shouldSend := lastPayload == "" || string(data) != lastPayload
		if !shouldSend && time.Since(lastSent) > 15*time.Second {
			shouldSend = true
		}

		if shouldSend {
			c.SSEvent("sessions", payload)
			c.Writer.Flush()
			lastPayload = string(data)
			lastSent = time.Now()
		}
	}

	send()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			send()
		}
	}
}
