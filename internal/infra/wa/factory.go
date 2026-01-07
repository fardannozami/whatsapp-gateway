package wa

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.mau.fi/whatsmeow/store/sqlstore"
	_ "modernc.org/sqlite"
)

func NewSQLStoreContainer(sqlPath string) (*sqlstore.Container, error) {
	container, _, err := OpenSQLStore(sqlPath)
	return container, err
}

func OpenSQLStore(sqlPath string) (*sqlstore.Container, *sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(sqlPath), 0o755); err != nil {
		return nil, nil, fmt.Errorf("mkdir db dir: %w", err)
	}

	db, err := sql.Open("sqlite", sqliteDSN(sqlPath))
	if err != nil {
		return nil, nil, fmt.Errorf("Open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if _, err := db.ExecContext(context.Background(), "PRAGMA foreign_keys = ON;"); err != nil {
		return nil, nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	container := sqlstore.NewWithDB(db, "sqlite", nil)
	if err := container.Upgrade(context.Background()); err != nil {
		return nil, nil, fmt.Errorf("upgrade db schema: %w", err)
	}

	return container, db, nil
}

func sqliteDSN(path string) string {
	params := "_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	if strings.Contains(path, "?") {
		return path + "&" + params
	}
	return path + "?" + params
}
