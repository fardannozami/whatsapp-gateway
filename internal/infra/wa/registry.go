package wa

import "go.mau.fi/whatsmeow"

func (m *Manager) addClient(JID string, client *whatsmeow.Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.clients[JID] = client
}

func (m *Manager) getClient(JID string) *whatsmeow.Client {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.clients[JID]
}

func (m *Manager) removeClient(JID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.clients, JID)
}

func (m *Manager) addPendingClient(JID string, client *whatsmeow.Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.pending[JID] = client
}

func (m *Manager) getPendingClient(JID string) *whatsmeow.Client {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.pending[JID]
}

func (m *Manager) removePendingClient(JID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.pending, JID)
}

func (m *Manager) getAllClients() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]string, 0, len(m.clients))
	for JID := range m.clients {
		result = append(result, JID)
	}

	return result
}
