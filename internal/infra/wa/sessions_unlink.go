package wa

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.mau.fi/whatsmeow"
)

func (m *Manager) unlinkSession(ctx context.Context, session string) error {
	client, temp, err := m.clientForUnlink(session)
	if err != nil {
		return err
	}
	if client == nil {
		return nil
	}

	if client.Store.ID == nil {
		return nil
	}

	if !client.IsConnected() {
		if err := connectWithTimeout(ctx, client, 20*time.Second); err != nil {
			if temp {
				disconnectClient(client)
			}
			return err
		}
	}

	if err := logoutClient(ctx, client); err != nil {
		if temp {
			disconnectClient(client)
		}
		return err
	}

	if temp {
		disconnectClient(client)
	}

	return nil
}

func (m *Manager) clientForUnlink(session string) (*whatsmeow.Client, bool, error) {
	m.mu.RLock()
	if c, ok := m.clients[session]; ok {
		m.mu.RUnlock()
		return c, false, nil
	}
	m.mu.RUnlock()

	dbPath := dbPathForSession(m.dbBasePath, session)
	if _, err := os.Stat(dbPath); err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}

	container, err := m.getContainer(session)
	if err != nil {
		return nil, false, err
	}
	device, err := getDeviceFromContainer(container)
	if err != nil {
		return nil, false, err
	}
	client := whatsmeow.NewClient(device, m.log)
	m.registerEventHandlers(session, client)
	return client, true, nil
}

func connectWithTimeout(ctx context.Context, client *whatsmeow.Client, timeout time.Duration) error {
	if client.IsConnected() {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- client.Connect()
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func logoutClient(ctx context.Context, client *whatsmeow.Client) error {
	type logoutWithCtx interface {
		Logout(context.Context) error
	}
	type logoutErr interface {
		Logout() error
	}
	type logoutNoErr interface {
		Logout()
	}

	if l, ok := interface{}(client).(logoutWithCtx); ok {
		return l.Logout(ctx)
	}
	if l, ok := interface{}(client).(logoutErr); ok {
		return l.Logout()
	}
	if l, ok := interface{}(client).(logoutNoErr); ok {
		l.Logout()
		return nil
	}
	return fmt.Errorf("logout not supported")
}

func disconnectClient(client *whatsmeow.Client) {
	type disconnecter interface {
		Disconnect()
	}
	if d, ok := interface{}(client).(disconnecter); ok {
		d.Disconnect()
	}
}
