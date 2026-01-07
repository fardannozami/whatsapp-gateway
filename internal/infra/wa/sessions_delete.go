package wa

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.mau.fi/whatsmeow"
)

func (m *Manager) DeleteSession(ctx context.Context, session string) (bool, error) {
	return m.deleteSessionWithRetry(ctx, session, 5, 200*time.Millisecond, true)
}

func (m *Manager) DeleteSessionForce(ctx context.Context, session string) (bool, error) {
	return m.deleteSessionWithRetry(ctx, session, 40, 250*time.Millisecond, false)
}

func (m *Manager) deleteSessionWithRetry(ctx context.Context, session string, attempts int, delay time.Duration, unlinkStrict bool) (hadSession bool, err error) {
	if err = ctx.Err(); err != nil {
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

	if err := m.unlinkSession(ctx, key); err != nil {
		if unlinkStrict {
			return true, fmt.Errorf("unlink failed: %w", err)
		}
		log.Printf("unlink %s: %v", key, err)
	}

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
		disconnectClient(client)
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
