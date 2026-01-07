package wa

import (
	"context"
	"fmt"
	"log"
	"os"
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
	status     map[string]string
	pairMu     sync.RWMutex
	pairing    map[string]PairingState
}

func NewManager(dbBasePath string, logger walog.Logger) *Manager {
	return &Manager{
		dbBasePath: dbBasePath,
		log:        logger,
		clients:    make(map[string]*whatsmeow.Client),
		containers: make(map[string]*sqlstore.Container),
		status:     make(map[string]string),
		pairing:    make(map[string]PairingState),
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
	device, err := getDeviceFromContainer(container)
	if err != nil {
		return nil, nil, fmt.Errorf("load device: %w", err)
	}
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

func (m *Manager) AutoConnectExisting(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	sessions, err := listSessionsFromDisk(m.dbBasePath)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	for _, session := range sessions {
		session := session
		wg.Add(1)
		go func() {
			defer wg.Done()

			client, err := m.CreateOrGetClientBySession(session)
			if err != nil {
				m.setStatus(session, "failed")
				log.Printf("auto connect %s: %v", session, err)
				return
			}

			if err := m.EnsureConnected(ctx, session, client); err != nil {
				log.Printf("auto connect %s: %v", session, err)
			}
		}()
	}
	wg.Wait()

	return nil
}

func listSessionsFromDisk(basePath string) ([]string, error) {
	dir, prefix, err := sessionsDirPrefix(basePath)
	if err != nil {
		return nil, err
	}
	if dir == "" {
		return nil, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if filepath.Ext(name) != ".db" {
			continue
		}

		if prefix != "" && !strings.HasPrefix(name, prefix) {
			continue
		}

		base := strings.TrimSuffix(name, ".db")
		if prefix != "" {
			base = strings.TrimPrefix(base, prefix)
		}

		if base == "" || !sessionKeyRe.MatchString(base) {
			continue
		}

		out = append(out, base)
	}

	return out, nil
}

func sessionsDirPrefix(basePath string) (string, string, error) {
	if basePath == "" {
		return ".", "", nil
	}

	if filepath.Ext(basePath) != ".db" {
		info, err := os.Stat(basePath)
		if err != nil {
			if os.IsNotExist(err) {
				return "", "", nil
			}
			return "", "", err
		}

		if info.IsDir() {
			return basePath, "", nil
		}

		return "", "", nil
	}

	dir := filepath.Dir(basePath)
	base := strings.TrimSuffix(filepath.Base(basePath), ".db")

	info, err := os.Stat(basePath)
	if err != nil {
		if os.IsNotExist(err) {
			dirInfo, dirErr := os.Stat(dir)
			if dirErr == nil && dirInfo.IsDir() {
				return dir, base + "-", nil
			}
			if dirErr != nil && !os.IsNotExist(dirErr) {
				return "", "", dirErr
			}
			return "", "", nil
		}
		return "", "", err
	}

	if info.IsDir() {
		return basePath, "", nil
	}

	return dir, base + "-", nil
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

	if client.IsConnected() {
		return "working"
	}

	if status, ok := m.getStatus(key); ok {
		if status == "connecting" || status == "failed" {
			return status
		}
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
}

func (m *Manager) getStatus(session string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	status, ok := m.status[session]
	return status, ok
}

type PairingState struct {
	Phone       string
	Code        string
	IssuedAt    time.Time
	ExpiresAt   time.Time
	LastAttempt time.Time
	LastError   string
	NextRetryAt time.Time
}

func (m *Manager) SetPairingPhone(session, phone string) error {
	key, err := normalizeSession(session)
	if err != nil {
		return err
	}

	m.pairMu.Lock()
	defer m.pairMu.Unlock()

	state := m.pairing[key]
	if state.Phone != phone {
		state.Phone = phone
		state.Code = ""
		state.IssuedAt = time.Time{}
		state.ExpiresAt = time.Time{}
		state.LastAttempt = time.Time{}
		state.LastError = ""
		state.NextRetryAt = time.Time{}
	}
	m.pairing[key] = state
	return nil
}

func (m *Manager) UpdatePairingCode(session, code string, issuedAt time.Time, ttl time.Duration) error {
	key, err := normalizeSession(session)
	if err != nil {
		return err
	}

	m.pairMu.Lock()
	defer m.pairMu.Unlock()

	state := m.pairing[key]
	state.Code = code
	state.IssuedAt = issuedAt
	state.ExpiresAt = issuedAt.Add(ttl)
	state.LastAttempt = time.Time{}
	state.LastError = ""
	state.NextRetryAt = time.Time{}
	m.pairing[key] = state
	return nil
}

func (m *Manager) UpdatePairingFailure(session, errMsg string, at time.Time, backoff time.Duration) error {
	key, err := normalizeSession(session)
	if err != nil {
		return err
	}

	m.pairMu.Lock()
	defer m.pairMu.Unlock()

	state := m.pairing[key]
	state.LastAttempt = at
	state.LastError = errMsg
	if backoff > 0 {
		state.NextRetryAt = at.Add(backoff)
	} else {
		state.NextRetryAt = time.Time{}
	}
	m.pairing[key] = state
	return nil
}

func (m *Manager) GetPairingState(session string) (PairingState, bool) {
	key, err := normalizeSession(session)
	if err != nil {
		return PairingState{}, false
	}

	m.pairMu.RLock()
	defer m.pairMu.RUnlock()
	state, ok := m.pairing[key]
	return state, ok
}

func (m *Manager) ClearPairing(session string) error {
	key, err := normalizeSession(session)
	if err != nil {
		return err
	}

	m.pairMu.Lock()
	defer m.pairMu.Unlock()
	delete(m.pairing, key)
	return nil
}
