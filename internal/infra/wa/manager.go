package wa

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	walog "go.mau.fi/whatsmeow/util/log"
)

type Manager struct {
	dbBasePath string
	log        walog.Logger
	mu         sync.RWMutex
	clients    map[string]*whatsmeow.Client
	containers map[string]*sqlstore.Container
	dbs        map[string]*sql.DB
	status     map[string]string
	statusFile string
	statusMu   sync.Mutex
	pairMu     sync.RWMutex
	pairing    map[string]PairingState
}

func NewManager(dbBasePath string, logger walog.Logger) *Manager {
	m := &Manager{
		dbBasePath: dbBasePath,
		log:        logger,
		clients:    make(map[string]*whatsmeow.Client),
		containers: make(map[string]*sqlstore.Container),
		dbs:        make(map[string]*sql.DB),
		status:     make(map[string]string),
		statusFile: statusFilePath(dbBasePath),
		pairing:    make(map[string]PairingState),
	}
	m.loadPersistedStatuses()
	return m
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
	if status, ok := m.getStatus(key); ok && status == "deleting" {
		return nil, fmt.Errorf("session is being deleted")
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

	// 3️⃣ load device dari DB kalau ada, kalau belum ada bikin baru
	container, err := m.getContainerLocked(key)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}
	device, err := getDeviceFromContainer(container)
	if err != nil {
		return nil, fmt.Errorf("load device: %w", err)
	}

	// 4️⃣ create client
	client := whatsmeow.NewClient(device, m.log)
	m.registerEventHandlers(key, client)

	// 5️⃣ simpan client by session
	m.clients[key] = client

	return client, nil
}

func (m *Manager) CreateOrGetClientByPhone(phone string) (*whatsmeow.Client, *store.Device, error) {
	key, err := normalizeSession(phone)
	if err != nil {
		return nil, nil, err
	}
	if status, ok := m.getStatus(key); ok && status == "deleting" {
		return nil, nil, fmt.Errorf("session is being deleted")
	}
	container, err := m.getContainer(key)
	if err != nil {
		return nil, nil, fmt.Errorf("open store: %w", err)
	}
	device, err := getDeviceFromContainer(container)
	if err != nil {
		return nil, nil, fmt.Errorf("load device: %w", err)
	}
	client := whatsmeow.NewClient(device, m.log)
	m.registerEventHandlers(key, client)

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

func (m *Manager) EnsureConnected(ctx context.Context, session string, client *whatsmeow.Client) error {
	key, err := normalizeSession(session)
	if err != nil {
		return err
	}

	if client.IsConnected() {
		m.setStatus(key, "working")
		return nil
	}

	m.setStatus(key, "connecting")

	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- client.Connect()
	}()

	select {
	case <-ctx.Done():
		m.setStatus(key, "failed")
		return fmt.Errorf("connect timeout: %w", ctx.Err())
	case err := <-errCh:
		if err != nil {
			m.setStatus(key, "failed")
			return err
		}
		m.setStatus(key, "working")
		return err
	}
}

func (m *Manager) PairPhone(ctx context.Context, client *whatsmeow.Client, phone string) (string, error) {
	// showPushNotification = true biar lebih smooth
	code, err := client.PairPhone(ctx, phone, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
	if err != nil {
		log.Println("⚠️ PairPhone warning:", err)
		return "", err
	}
	if code == "" {
		return "", fmt.Errorf("pair phone returned empty code")
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

func (m *Manager) SessionStatus(session string, client *whatsmeow.Client) string {
	key, err := normalizeSession(session)
	if err != nil {
		return "failed"
	}

	if status, ok := m.getStatus(key); ok {
		if status == "logout" {
			return status
		}
		if status == "deleting" {
			return status
		}
		if status == "connecting" || status == "failed" {
			return status
		}
	}

	if client.IsConnected() {
		return "working"
	}

	if client.Store.ID != nil {
		return "stopped"
	}

	return "failed"
}

func (m *Manager) setStatus(session, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status[session] = status
	if status == "logout" {
		_ = m.persistStatus(session, status)
	}
}

func (m *Manager) getStatus(session string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	status, ok := m.status[session]
	return status, ok
}

func (m *Manager) registerEventHandlers(session string, client *whatsmeow.Client) {
	client.AddEventHandler(func(evt interface{}) {
		switch evt.(type) {
		case *events.LoggedOut:
			m.setStatus(session, "logout")
		}
	})
}
