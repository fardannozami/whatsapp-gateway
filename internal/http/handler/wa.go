package handler

import (
	"net/http"

	"github.com/fardannozami/whatsapp-gateway/internal/http/dto"
	"github.com/fardannozami/whatsapp-gateway/internal/infra/wa"
	"github.com/gin-gonic/gin"
)

type WaHandler struct {
	manager *wa.Manager
}

func NewWaHandler(manager *wa.Manager) *WaHandler {
	return &WaHandler{
		manager: manager,
	}
}

func (h *WaHandler) BeginPair(c *gin.Context) {
	var req dto.PairRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pairingId, pairingCode, err := h.manager.BeginPair(c.Request.Context(), req.Phone)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
	}

	c.JSON(http.StatusOK, dto.PairResponse{
		PairingId:   pairingId,
		PairingCode: pairingCode,
	})
}
