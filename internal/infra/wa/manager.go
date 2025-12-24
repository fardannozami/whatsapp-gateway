package wa

import (
	"sync"

	"go.mau.fi/whatsmeow"
	walog "go.mau.fi/whatsmeow/util/log"

	"go.mau.fi/whatsmeow/store/sqlstore"
)

type Manager struct {
	container sqlstore.Container
	log       walog.Logger

	mu sync.RWMutex

	clients map[string]*whatsmeow.Client
	pending map[string]*whatsmeow.Client
}

func NewManager(container sqlstore.Container) *Manager {
	return &Manager{
		container: container,
		log:       walog.Stdout("WAManager", "DEBUG", true),
		clients:   make(map[string]*whatsmeow.Client),
		pending:   make(map[string]*whatsmeow.Client),
	}
}
