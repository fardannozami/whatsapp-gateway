package wa

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.mau.fi/whatsmeow"
)

func (m *Manager) BeginPair(ctx context.Context, phone string) (pairingId, pairingCode string, err error) {
	dev := m.container.NewDevice()

	client := whatsmeow.NewClient(dev, m.log)
	m.attachHandler(client)

	pairingCode, err = client.PairPhone(ctx, phone, true, whatsmeow.PairClientChrome, "WA Gateway")
	if err != nil {
		return "", "", err
	}

	pairingId = uuid.NewString()
	m.addPendingClient(pairingId, client)

	go m.waitForLogin(pairingId, client)

	return pairingId, pairingCode, nil
}

func (m *Manager) waitForLogin(pairingId string, client *whatsmeow.Client) {
	timeout := time.After(90 * time.Second)
	tick := time.NewTicker(1 * time.Second)

	defer tick.Stop()

	for {
		select {
		case <-timeout:
			m.removePendingClient(pairingId)
			client.Disconnect()
			m.log.Warnf("pairing %s timed out", pairingId)
			return

		case <-tick.C:
			if client.Store.ID != nil && client.Store.ID.User != "" {
				m.removePendingClient(pairingId)
				jid := client.Store.ID.String()
				m.addClient(jid, client)
				m.log.Infof("pairing success: %s -> %s", pairingId, jid)
				return
			}
		}
	}
}
