package wa

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	walog "go.mau.fi/whatsmeow/util/log"
)

type Manager struct {
	dbBasePath string
	log        walog.Logger
	mu         sync.RWMutex
	clients    map[string]*whatsmeow.Client
	containers map[string]*sqlstore.Container
}

func NewManager(dbBasePath string, logger walog.Logger) *Manager {
	return &Manager{
		dbBasePath: dbBasePath,
		log:        logger,
		clients:    make(map[string]*whatsmeow.Client),
		containers: make(map[string]*sqlstore.Container),
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

func (m *Manager) CreateOrGetClientBySession(session string) (*whatsmeow.Client, error) {
	key, err := normalizeSession(session)
	if err != nil {
		return nil, err
	}

	// 1️⃣ cek memory
	m.mu.RLock()
	if c, ok := m.clients[key]; ok {
		m.mu.RUnlock()
		return c, nil
	}
	m.mu.RUnlock()

	// 2️⃣ lock create (anti race)
	m.mu.Lock()
	defer m.mu.Unlock()

	if c, ok := m.clients[key]; ok {
		return c, nil
	}

	// 3️⃣ SELALU NewDevice
	// sqlstore akan auto-load device lama kalau ada
	container, err := m.getContainerLocked(key)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}
	device := container.NewDevice()

	// 4️⃣ create client
	client := whatsmeow.NewClient(device, m.log)

	// 5️⃣ simpan client by session
	m.clients[key] = client

	return client, nil
}

func (m *Manager) CreateOrGetClientByPhone(phone string) (*whatsmeow.Client, *store.Device, error) {
	key, err := normalizeSession(phone)
	if err != nil {
		return nil, nil, err
	}
	container, err := m.getContainer(key)
	if err != nil {
		return nil, nil, fmt.Errorf("open store: %w", err)
	}
	device := container.NewDevice()
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

var sessionKeyRe = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func normalizeSession(session string) (string, error) {
	if session == "" {
		return "", fmt.Errorf("session is required")
	}
	if !sessionKeyRe.MatchString(session) {
		return "", fmt.Errorf("invalid session: use letters, numbers, dash, underscore")
	}
	return session, nil
}

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

func (m *Manager) PairPhone(ctx context.Context, client *whatsmeow.Client, phone string) (string, error) {
	// showPushNotification = true biar lebih smooth
	code, err := client.PairPhone(ctx, phone, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
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
