package http

import (
	"net/http"

	"github.com/fardannozami/whatsapp-gateway/internal/app/usecase"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	pairUC *usecase.PairCodeUsecase
	listUC *usecase.ListClientsUsecase
}

func NewHandler(pairUC *usecase.PairCodeUsecase, listUC *usecase.ListClientsUsecase) *Handler {
	return &Handler{pairUC: pairUC, listUC: listUC}
}

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

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
		c.JSON(http.StatusBadRequest, gin.H{"error": "pair failed", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, PairCodeResponse{
		Status:      out.Status,
		PairingCode: out.PairingCode,
	})
}

func (h *Handler) Clients(c *gin.Context) {
	clients := h.listUC.Execute()
	c.JSON(http.StatusOK, ClientsResponse{
		Count:   len(clients),
		Clients: clients,
	})
}
