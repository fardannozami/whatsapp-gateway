package config

type Config struct {
	HTTPAddr string
	DB       struct {
		Driver string // sqlite | mysql | postgres
		DSN    string
	}
}

func Load() Config {
	var cfg Config

	cfg.HTTPAddr = ":8080"

	cfg.DB.Driver = "sqlite"
	cfg.DB.DSN = "file:./data/whatsmeow.db?_pragma=foreign_keys(1)"

	// nanti tinggal ganti:
	// mysql:    user:pass@tcp(localhost:3306)/dbname?parseTime=true
	// postgres: postgres://user:pass@localhost:5432/dbname?sslmode=disable

	return cfg
}
