package http

import "github.com/gin-gonic/gin"

func NewRouter(h *Handler) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.GET("/health", h.Health)

	wa := r.Group("/api")
	wa.POST("/:session/auth/request-code", h.PairCode)
	wa.GET("/clients", h.Clients)

	return r
}
