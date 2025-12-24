package wa

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow/store/sqlstore"
	walog "go.mau.fi/whatsmeow/util/log"
)

func NewSQLStore(ctx context.Context, driver, dsn string) (*sqlstore.Container, error) {
	walog := walog.Stdout("Database", "DEBUG", true)

	switch driver {
	case "sqlite", "mysql", "postgres":
		return sqlstore.New(ctx, driver, dsn, walog)

	default:
		return nil, fmt.Errorf("unsupported db driver: %s", driver)
	}
}
