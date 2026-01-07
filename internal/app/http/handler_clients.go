package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) Clients(c *gin.Context) {
	clients := h.listUC.Execute()
	c.JSON(http.StatusOK, ClientsResponse{
		Count:   len(clients),
		Clients: clients,
	})
}
