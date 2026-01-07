package wa

import "time"

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

	_ = m.clearPersistedStatus(key)

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

	_ = m.clearPersistedStatus(key)

	m.pairMu.Lock()
	defer m.pairMu.Unlock()
	delete(m.pairing, key)
	return nil
}
