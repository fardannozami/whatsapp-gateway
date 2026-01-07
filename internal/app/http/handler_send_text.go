package http

import (
	"net/http"
	"strings"

	"github.com/fardannozami/whatsapp-gateway/internal/app/usecase"
	"github.com/gin-gonic/gin"
)

func (h *Handler) SendText(c *gin.Context) {
	session := c.Param("session")
	if session == "" {
		c.JSON(400, gin.H{
			"error": "session param is required",
		})
		return
	}

	var req SendTextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json", "detail": err.Error()})
		return
	}

	req.Message = strings.TrimSpace(req.Message)
	if req.To == "" || req.Message == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "to and message are required"})
		return
	}

	out, err := h.sendUC.Execute(c.Request.Context(), usecase.SendTextInput{
		Session: session,
		To:      req.To,
		Message: req.Message,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "send text failed", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, SendTextResponse{
		Status:    out.Status,
		MessageID: out.MessageID,
	})
}
