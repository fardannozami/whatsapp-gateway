package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/fardannozami/whatsapp-gateway/internal/app/usecase"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	pairUC *usecase.PairCodeUsecase
	listUC *usecase.ListClientsUsecase
	meUC   *usecase.MeUsecase
	pairSU *usecase.PairStreamUsecase
	sessUC *usecase.ListSessionsUsecase
}

func NewHandler(pairUC *usecase.PairCodeUsecase, listUC *usecase.ListClientsUsecase, meUC *usecase.MeUsecase, pairSU *usecase.PairStreamUsecase, sessUC *usecase.ListSessionsUsecase) *Handler {
	return &Handler{pairUC: pairUC, listUC: listUC, meUC: meUC, pairSU: pairSU, sessUC: sessUC}
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
		errMsg := err.Error()

		// detect rate limit dari WhatsApp
		if strings.Contains(errMsg, "429") ||
			strings.Contains(errMsg, "rate-overlimit") {

			c.JSON(http.StatusTooManyRequests, gin.H{
				"status": "too_many",
				"error":  "pair failed",
				"detail": errMsg,
			})
			return
		}

		// default error
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "failed",
			"error":  "pair failed",
			"detail": errMsg,
		})
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

func (h *Handler) PairStream(c *gin.Context) {
	session := c.Param("session")
	if session == "" {
		c.JSON(400, gin.H{
			"error": "session param is required",
		})
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()

	ctx := c.Request.Context()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	lastStatus := ""
	lastCode := ""

	send := func() (bool, error) {
		out, done, err := h.pairSU.Next(ctx, session)
		if err != nil {
			payload := PairStreamResponse{
				Status: "failed",
				Detail: err.Error(),
			}
			c.SSEvent("pair", payload)
			c.Writer.Flush()
			return done, err
		}
		if out == nil {
			return done, nil
		}

		if out.Status != lastStatus || out.PairingCode != lastCode {
			payload := PairStreamResponse{
				Status:      out.Status,
				PairingCode: out.PairingCode,
				ExpiresIn:   out.ExpiresIn,
				RetryIn:     out.RetryIn,
				Detail:      out.Detail,
			}
			c.SSEvent("pair", payload)
			c.Writer.Flush()
			lastStatus = out.Status
			lastCode = out.PairingCode
		}

		return done, nil
	}

	if done, _ := send(); done {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if done, _ := send(); done {
				return
			}
		}
	}
}

func (h *Handler) SessionsStream(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()

	ctx := c.Request.Context()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	lastPayload := ""
	lastSent := time.Time{}

	send := func() {
		items, err := h.sessUC.Execute(ctx)
		if err != nil {
			payload := SessionsStreamResponse{
				Status: "failed",
				Detail: err.Error(),
			}
			c.SSEvent("sessions", payload)
			c.Writer.Flush()
			lastSent = time.Now()
			return
		}

		sessions := make([]SessionItemResponse, 0, len(items))
		for _, item := range items {
			sessions = append(sessions, SessionItemResponse{
				Session:  item.Session,
				ID:       item.ID,
				PushName: item.PushName,
				Status:   item.Status,
			})
		}

		payload := SessionsStreamResponse{
			Status:   "ok",
			Sessions: sessions,
		}

		data, err := json.Marshal(payload)
		if err != nil {
			return
		}

		shouldSend := lastPayload == "" || string(data) != lastPayload
		if !shouldSend && time.Since(lastSent) > 15*time.Second {
			shouldSend = true
		}

		if shouldSend {
			c.SSEvent("sessions", payload)
			c.Writer.Flush()
			lastPayload = string(data)
			lastSent = time.Now()
		}
	}

	send()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			send()
		}
	}
}

func (h *Handler) Clients(c *gin.Context) {
	clients := h.listUC.Execute()
	c.JSON(http.StatusOK, ClientsResponse{
		Count:   len(clients),
		Clients: clients,
	})
}
