package http

import (
	"github.com/fardannozami/whatsapp-gateway/internal/http/handler"
	"github.com/fardannozami/whatsapp-gateway/internal/infra/wa"
	"github.com/gin-gonic/gin"
)

func NewRouter(m *wa.Manager) *gin.Engine {
	r := gin.Default()
	h := handler.NewWaHandler(m)

	r.POST("/pair", h.BeginPair)

	return r
}
