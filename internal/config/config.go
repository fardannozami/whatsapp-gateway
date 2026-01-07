package config

import "os"

type Config struct {
	Port       string
	SQLitePath string
}

func Load() Config {
	port := getenv("PORT", "8080")
	sqlitePath := getenv("SQLITE_PATH", "./data/whatsapp.db")

	return Config{
		Port:       port,
		SQLitePath: sqlitePath,
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}
