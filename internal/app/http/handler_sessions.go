package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) DeleteSession(c *gin.Context) {
	session := c.Param("session")
	if session == "" {
		c.JSON(400, gin.H{
			"error": "session param is required",
		})
		return
	}

	deleted, err := h.delUC.Execute(c.Request.Context(), session)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "delete session failed", "detail": err.Error()})
		return
	}
	if !deleted {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	c.JSON(http.StatusOK, DeleteSessionResponse{Status: "deleted"})
}

func (h *Handler) ForceDeleteSession(c *gin.Context) {
	session := c.Param("session")
	if session == "" {
		c.JSON(400, gin.H{
			"error": "session param is required",
		})
		return
	}

	deleted, err := h.delFUC.Execute(c.Request.Context(), session)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "force delete session failed", "detail": err.Error()})
		return
	}
	if !deleted {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	c.JSON(http.StatusOK, DeleteSessionResponse{Status: "deleted"})
}

func (h *Handler) StopSession(c *gin.Context) {
	session := c.Param("session")
	if session == "" {
		c.JSON(400, gin.H{
			"error": "session param is required",
		})
		return
	}

	stopped, err := h.stopUC.Execute(c.Request.Context(), session)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "stop session failed", "detail": err.Error()})
		return
	}
	if !stopped {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	c.JSON(http.StatusOK, StopSessionResponse{Status: "stopped"})
}
