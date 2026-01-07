package wa

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	walog "go.mau.fi/whatsmeow/util/log"
)

type Manager struct {
	container *sqlstore.Container
	log       walog.Logger
	mu        sync.RWMutex
	clients   map[string]*whatsmeow.Client
}

func NewManager(container *sqlstore.Container, logger walog.Logger) *Manager {
	return &Manager{
		container: container,
		log:       logger,
		clients:   make(map[string]*whatsmeow.Client),
	}
}

func (m *Manager) ListClients() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]string, 0, len(m.clients))
	for jid := range m.clients {
		out = append(out, jid)
	}

	return out
}

func (m *Manager) GetClient(jid string) (*whatsmeow.Client, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.clients[jid]
	return c, ok
}

func (m *Manager) CreateOrGetClientByPhone(phone string) (*whatsmeow.Client, *store.Device, error) {
	device := m.container.NewDevice()
	client := whatsmeow.NewClient(device, m.log)

	return client, device, nil
}

func (m *Manager) RegisterClientInMemory(client *whatsmeow.Client) error {
	jid := client.Store.ID
	if jid == nil {
		return fmt.Errorf("client has no jid yet")
	}

	key := jid.String()

	m.mu.Lock()
	defer m.mu.Unlock()
	m.clients[key] = client

	return nil
}

func (m *Manager) EnsureConnected(ctx context.Context, client *whatsmeow.Client) error {
	if client.IsConnected() {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- client.Connect()
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("connect timeout: %w", ctx.Err())
	case err := <-errCh:
		return err
	}
}

func ParseClientType(s string) whatsmeow.PairClientType {
	switch s {
	case "chrome":
		return whatsmeow.PairClientChrome
	case "firefox":
		return whatsmeow.PairClientFirefox
	case "safari":
		return whatsmeow.PairClientSafari
	case "edge":
		return whatsmeow.PairClientEdge
	default:
		return whatsmeow.PairClientChrome
	}
}

func (m *Manager) PairPhone(ctx context.Context, client *whatsmeow.Client, phone string, clientType whatsmeow.PairClientType) (string, error) {
	// showPushNotification = true biar lebih smooth
	fmt.Println(clientType)
	code, err := client.PairPhone(ctx, phone, true, clientType, "Chrome (Linux)")
	if err != nil {
		log.Println("⚠️ PairPhone warning:", err)
	}

	fmt.Println("🔢 Pairing Code:", code)
	fmt.Println("➡️ WhatsApp > Linked Devices > Link with phone number")
	return code, nil
}

// AfterPairing, on successful login, client.Store.ID akan terisi types.JID
func (m *Manager) GetJIDString(client *whatsmeow.Client) (string, bool) {
	id := client.Store.ID
	if id == nil {
		return "", false
	}
	return types.JID(*id).String(), true
}
