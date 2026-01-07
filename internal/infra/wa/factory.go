package wa

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"go.mau.fi/whatsmeow/store/sqlstore"
	_ "modernc.org/sqlite"
)

func NewSQLStoreContainer(sqlPath string) (*sqlstore.Container, error) {
	if err := os.MkdirAll(filepath.Dir(sqlPath), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir db dir: %w", err)
	}

	db, err := sql.Open("sqlite", sqlPath)
	if err != nil {
		return nil, fmt.Errorf("Open sqlite: %w", err)
	}

	container := sqlstore.NewWithDB(db, "sqlite", nil)

	return container, nil
}
