package wa

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
)

type SessionInfo struct {
	Session  string
	ID       string
	PushName string
	Status   string
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
		if status, ok := m.getStatus(session); ok && status == "logout" {
			continue
		}
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

func (m *Manager) ListSessions(ctx context.Context) ([]SessionInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	sessions, err := m.collectSessions()
	if err != nil {
		return nil, err
	}

	out := make([]SessionInfo, 0, len(sessions))
	for _, session := range sessions {
		info := SessionInfo{Session: session}
		if status, ok := m.getStatus(session); ok && status == "deleting" {
			info.Status = "deleting"
			out = append(out, info)
			continue
		}

		client, err := m.CreateOrGetClientBySession(session)
		if err != nil {
			info.Status = "failed"
			out = append(out, info)
			continue
		}

		id := ""
		if client.Store.ID != nil {
			id = client.Store.ID.User
		}

		info.ID = id
		info.PushName = client.Store.PushName
		info.Status = m.SessionStatus(session, client)
		out = append(out, info)
	}

	return out, nil
}

func (m *Manager) collectSessions() ([]string, error) {
	set := make(map[string]struct{})

	diskSessions, err := listSessionsFromDisk(m.dbBasePath)
	if err != nil {
		return nil, err
	}
	for _, session := range diskSessions {
		set[session] = struct{}{}
	}

	m.pairMu.RLock()
	for session := range m.pairing {
		set[session] = struct{}{}
	}
	m.pairMu.RUnlock()

	m.mu.RLock()
	for session := range m.clients {
		if sessionKeyRe.MatchString(session) {
			set[session] = struct{}{}
		}
	}
	m.mu.RUnlock()

	out := make([]string, 0, len(set))
	for session := range set {
		out = append(out, session)
	}
	sort.Strings(out)
	return out, nil
}

func (m *Manager) DeleteSession(ctx context.Context, session string) (bool, error) {
	return m.deleteSessionWithRetry(ctx, session, 5, 200*time.Millisecond)
}

func (m *Manager) DeleteSessionForce(ctx context.Context, session string) (bool, error) {
	return m.deleteSessionWithRetry(ctx, session, 40, 250*time.Millisecond)
}

func (m *Manager) deleteSessionWithRetry(ctx context.Context, session string, attempts int, delay time.Duration) (hadSession bool, err error) {
	if err = ctx.Err(); err != nil {
		return false, err
	}

	key, err := normalizeSession(session)
	if err != nil {
		return false, err
	}

	prevStatus, prevOK := m.getStatus(key)
	m.setStatus(key, "deleting")
	defer func() {
		if err == nil {
			return
		}
		if prevOK {
			m.setStatus(key, prevStatus)
			return
		}
		m.mu.Lock()
		delete(m.status, key)
		m.mu.Unlock()
	}()

	var client *whatsmeow.Client

	m.mu.Lock()
	if c, ok := m.clients[key]; ok {
		client = c
		hadSession = true
		delete(m.clients, key)
	}
	if _, ok := m.containers[key]; ok {
		hadSession = true
		delete(m.containers, key)
	}
	if db, ok := m.dbs[key]; ok {
		hadSession = true
		delete(m.dbs, key)
		_ = db.Close()
	}
	if _, ok := m.status[key]; ok {
		hadSession = true
	}
	m.mu.Unlock()

	m.pairMu.Lock()
	if _, ok := m.pairing[key]; ok {
		hadSession = true
		delete(m.pairing, key)
	}
	m.pairMu.Unlock()

	_ = m.clearPersistedStatus(key)

	if client != nil {
		type disconnecter interface {
			Disconnect()
		}
		if d, ok := interface{}(client).(disconnecter); ok {
			d.Disconnect()
		}
	}

	dbPath := dbPathForSession(m.dbBasePath, key)
	paths := []string{
		dbPath,
		dbPath + "-wal",
		dbPath + "-shm",
	}

	for _, path := range paths {
		removed, err := removeFileWithRetry(path, attempts, delay)
		if err != nil {
			return hadSession, err
		}
		if removed {
			hadSession = true
		}
	}

	m.mu.Lock()
	delete(m.status, key)
	m.mu.Unlock()

	return hadSession, nil
}

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
		type disconnecter interface {
			Disconnect()
		}
		if d, ok := interface{}(client).(disconnecter); ok {
			d.Disconnect()
		}
	}

	if status != "logout" && status != "deleting" {
		m.setStatus(key, "stopped")
	}

	return true, nil
}

func removeFileIfExists(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if err := os.Remove(path); err != nil {
		return false, err
	}
	return true, nil
}

func removeFileWithRetry(path string, attempts int, delay time.Duration) (bool, error) {
	var lastErr error
	for i := 0; i < attempts; i++ {
		removed, err := removeFileIfExists(path)
		if err == nil {
			return removed, nil
		}
		lastErr = err
		time.Sleep(delay)
	}
	return false, lastErr
}

func (m *Manager) sessionExists(session string) (bool, error) {
	m.mu.RLock()
	if _, ok := m.clients[session]; ok {
		m.mu.RUnlock()
		return true, nil
	}
	if _, ok := m.containers[session]; ok {
		m.mu.RUnlock()
		return true, nil
	}
	if _, ok := m.status[session]; ok {
		m.mu.RUnlock()
		return true, nil
	}
	m.mu.RUnlock()

	m.pairMu.RLock()
	if _, ok := m.pairing[session]; ok {
		m.pairMu.RUnlock()
		return true, nil
	}
	m.pairMu.RUnlock()

	dbPath := dbPathForSession(m.dbBasePath, session)
	if _, err := os.Stat(dbPath); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
