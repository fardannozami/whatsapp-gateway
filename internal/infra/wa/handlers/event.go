package handlers

import (
	"go.mau.fi/whatsmeow/types/events"
	walog "go.mau.fi/whatsmeow/util/log"
)

type EventHandler struct {
	log walog.Logger
}

func NewEventHandler(log walog.Logger) func(evt interface{}) {
	h := &EventHandler{log: log}
	return h.Handle
}

func (h *EventHandler) Handle(evt interface{}) {
	switch e := evt.(type) {

	case *events.Message:
		h.log.Infof(
			"incoming message from %s: %s",
			e.Info.Sender.String(),
			e.Message.GetConversation(),
		)

	case *events.Connected:
		h.log.Infof("client connected")

	case *events.Disconnected:
		h.log.Warnf("client disconnected")

	}
}
