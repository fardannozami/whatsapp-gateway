package wa

import (
	"context"

	"go.mau.fi/whatsmeow"
)

func (m *Manager) StopSession(ctx context.Context, session string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	key, err := normalizeSession(session)
	if err != nil {
		return false, err
	}

	exists, err := m.sessionExists(key)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}

	status, _ := m.getStatus(key)

	var client *whatsmeow.Client
	m.mu.Lock()
	if c, ok := m.clients[key]; ok {
		client = c
	}
	m.mu.Unlock()

	if client != nil {
		disconnectClient(client)
	}

	if status != "logout" && status != "deleting" {
		m.setStatus(key, "stopped")
	}

	return true, nil
}
