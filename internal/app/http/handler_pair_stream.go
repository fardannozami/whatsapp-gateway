package http

import (
	"time"

	"github.com/gin-gonic/gin"
)

func (h *Handler) PairStream(c *gin.Context) {
	session := c.Param("session")
	if session == "" {
		c.JSON(400, gin.H{
			"error": "session param is required",
		})
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()

	ctx := c.Request.Context()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	lastStatus := ""
	lastCode := ""

	send := func() (bool, error) {
		out, done, err := h.pairSU.Next(ctx, session)
		if err != nil {
			payload := PairStreamResponse{
				Status: "failed",
				Detail: err.Error(),
			}
			c.SSEvent("pair", payload)
			c.Writer.Flush()
			return done, err
		}
		if out == nil {
			return done, nil
		}

		if out.Status != lastStatus || out.PairingCode != lastCode {
			payload := PairStreamResponse{
				Status:      out.Status,
				PairingCode: out.PairingCode,
				ExpiresIn:   out.ExpiresIn,
				RetryIn:     out.RetryIn,
				Detail:      out.Detail,
			}
			c.SSEvent("pair", payload)
			c.Writer.Flush()
			lastStatus = out.Status
			lastCode = out.PairingCode
		}

		return done, nil
	}

	if done, _ := send(); done {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if done, _ := send(); done {
				return
			}
		}
	}
}
