package wa

import (
	"context"
	"path/filepath"
	"strings"

	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
)

func (m *Manager) getContainer(session string) (*sqlstore.Container, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.getContainerLocked(session)
}

func (m *Manager) getContainerLocked(session string) (*sqlstore.Container, error) {
	if c, ok := m.containers[session]; ok {
		return c, nil
	}

	path := dbPathForSession(m.dbBasePath, session)
	container, err := NewSQLStoreContainer(path)
	if err != nil {
		return nil, err
	}
	m.containers[session] = container
	return container, nil
}

func dbPathForSession(basePath, session string) string {
	if basePath == "" {
		return session + ".db"
	}

	if filepath.Ext(basePath) == ".db" {
		dir := filepath.Dir(basePath)
		base := strings.TrimSuffix(filepath.Base(basePath), ".db")
		filename := base + "-" + session + ".db"
		return filepath.Join(dir, filename)
	}

	return filepath.Join(basePath, session+".db")
}

func getDeviceFromContainer(container *sqlstore.Container) (*store.Device, error) {
	device, err := container.GetFirstDevice(context.Background())
	if err != nil {
		return nil, err
	}
	if device == nil {
		device = container.NewDevice()
	}
	return device, nil
}
