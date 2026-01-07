package wa

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
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
