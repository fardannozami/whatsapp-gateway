package wa

import (
	"context"

	"go.mau.fi/whatsmeow"
)

func (m *Manager) StartAll(ctx context.Context) error {
	devices, err := m.container.GetAllDevices(ctx)
	if err != nil {
		return err
	}

	for _, device := range devices {
		client := whatsmeow.NewClient(device, m.log)

		m.attachHandler(client)

		go func(c *whatsmeow.Client) {
			if err := c.Connect(); err != nil {
				m.log.Errorf("failed to connect client: %v", err)
				return
			}

			jid := c.Store.ID.String()
			m.addClient(jid, c)
			m.log.Infof("client %s connected", jid)
		}(client)
	}

	return nil
}
