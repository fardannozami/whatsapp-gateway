package http

import (
	"net/http"

	"github.com/fardannozami/whatsapp-gateway/internal/app/usecase"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	pairUC *usecase.PairCodeUsecase
	listUC *usecase.ListClientsUsecase
	meUC   *usecase.MeUsecase
}

func NewHandler(pairUC *usecase.PairCodeUsecase, listUC *usecase.ListClientsUsecase, meUC *usecase.MeUsecase) *Handler {
	return &Handler{pairUC: pairUC, listUC: listUC, meUC: meUC}
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

func (h *Handler) Clients(c *gin.Context) {
	clients := h.listUC.Execute()
	c.JSON(http.StatusOK, ClientsResponse{
		Count:   len(clients),
		Clients: clients,
	})
}
