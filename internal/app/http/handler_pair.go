package http

import (
	"net/http"
	"strings"

	"github.com/fardannozami/whatsapp-gateway/internal/app/usecase"
	"github.com/gin-gonic/gin"
)

func (h *Handler) PairCode(c *gin.Context) {
	session := c.Param("session")

	if session == "" {
		c.JSON(400, gin.H{
			"error": "session param is required",
		})
		return
	}

	var req PairCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json", "detail": err.Error()})
		return
	}

	out, err := h.pairUC.Execute(c.Request.Context(), usecase.PairCodeInput{
		Phone:   req.Phone,
		Session: session,
	})

	if err != nil {
		errMsg := err.Error()

		// detect rate limit dari WhatsApp
		if strings.Contains(errMsg, "429") ||
			strings.Contains(errMsg, "rate-overlimit") {

			c.JSON(http.StatusTooManyRequests, gin.H{
				"status": "too_many",
				"error":  "pair failed",
				"detail": errMsg,
			})
			return
		}

		// default error
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "failed",
			"error":  "pair failed",
			"detail": errMsg,
		})
		return
	}

	c.JSON(http.StatusOK, PairCodeResponse{
		Status:      out.Status,
		PairingCode: out.PairingCode,
	})
}
