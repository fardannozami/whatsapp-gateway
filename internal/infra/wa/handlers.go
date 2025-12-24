package wa

import (
	"github.com/fardannozami/whatsapp-gateway/internal/infra/wa/handlers"
	"go.mau.fi/whatsmeow"
)

func (m *Manager) attachHandler(client *whatsmeow.Client) {
	client.AddEventHandler(handlers.NewEventHandler(m.log))
}
