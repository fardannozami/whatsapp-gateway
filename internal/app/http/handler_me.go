package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) Me(c *gin.Context) {
	session := c.Param("session")

	if session == "" {
		c.JSON(400, gin.H{
			"error": "session param is required",
		})
		return
	}

	out, err := h.meUC.Execute(c.Request.Context(), session)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "get me failed", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, MeResponse{
		Status:   out.Status,
		Id:       out.ID,
		JID:      out.JID,
		PushName: out.PushName,
	})
}
