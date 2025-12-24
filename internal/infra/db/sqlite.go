package db

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

func Open(dsn string) (*sql.DB, error) {
	return sql.Open("sqlite", dsn)
}
