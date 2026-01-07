package http

import "github.com/gin-gonic/gin"

func NewRouter(h *Handler) *gin.Engine {
	r := gin.New()
	r.Use(RequestID(), RequestLogger(), gin.Recovery())

	r.GET("/health", h.Health)

	wa := r.Group("/api")
	wa.POST("/:session/auth/request-code", h.PairCode)
	wa.GET("/clients", h.Clients)

	sessions := wa.Group("/sessions")
	sessions.GET("/:session/me", h.Me)
	sessions.GET("/:session/pair/stream", h.PairStream)
	sessions.GET("/stream", h.SessionsStream)
	sessions.DELETE("/:session", h.DeleteSession)
	sessions.DELETE("/:session/force", h.ForceDeleteSession)
	sessions.POST("/:session/stop", h.StopSession)

	return r
}
