package wa

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type statusFileData struct {
	Sessions map[string]string `json:"sessions"`
}

func statusFilePath(basePath string) string {
	if basePath == "" {
		return "session-status.json"
	}
	if filepath.Ext(basePath) == ".db" {
		return filepath.Join(filepath.Dir(basePath), "session-status.json")
	}
	return filepath.Join(basePath, "session-status.json")
}

func (m *Manager) loadPersistedStatuses() {
	if m.statusFile == "" {
		return
	}

	data, err := os.ReadFile(m.statusFile)
	if err != nil {
		return
	}

	var state statusFileData
	if err := json.Unmarshal(data, &state); err != nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	for session, status := range state.Sessions {
		if status != "logout" {
			continue
		}
		if !sessionKeyRe.MatchString(session) {
			continue
		}
		m.status[session] = status
	}
}

func (m *Manager) persistStatus(session, status string) error {
	if m.statusFile == "" {
		return nil
	}
	if status != "logout" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(m.statusFile), 0o755); err != nil {
		return err
	}

	m.statusMu.Lock()
	defer m.statusMu.Unlock()

	var state statusFileData
	data, err := os.ReadFile(m.statusFile)
	if err == nil {
		_ = json.Unmarshal(data, &state)
	}
	if state.Sessions == nil {
		state.Sessions = make(map[string]string)
	}
	state.Sessions[session] = status

	out, err := json.Marshal(state)
	if err != nil {
		return err
	}

	tmp := m.statusFile + ".tmp"
	if err := os.WriteFile(tmp, out, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, m.statusFile)
}

func (m *Manager) clearPersistedStatus(session string) error {
	if m.statusFile == "" {
		return nil
	}

	m.statusMu.Lock()
	defer m.statusMu.Unlock()

	data, err := os.ReadFile(m.statusFile)
	if err != nil {
		return nil
	}

	var state statusFileData
	if err := json.Unmarshal(data, &state); err != nil {
		return nil
	}
	if state.Sessions == nil {
		return nil
	}
	if _, ok := state.Sessions[session]; !ok {
		return nil
	}

	delete(state.Sessions, session)
	out, err := json.Marshal(state)
	if err != nil {
		return err
	}
	tmp := m.statusFile + ".tmp"
	if err := os.WriteFile(tmp, out, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, m.statusFile)
}
