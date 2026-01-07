package main

import (
	"log"

	"github.com/fardannozami/whatsapp-gateway/internal/app/http"
	"github.com/fardannozami/whatsapp-gateway/internal/app/usecase"
	"github.com/fardannozami/whatsapp-gateway/internal/config"
	"github.com/fardannozami/whatsapp-gateway/internal/infra/wa"
	walog "go.mau.fi/whatsmeow/util/log"
)

func main() {
	cfg := config.Load()

	waLogger := walog.Stdout("WA", "INFO", true)

	container, err := wa.NewSQLStoreContainer(cfg.SQLitePath)
	if err != nil {
		log.Fatal(err)
	}

	waManager := wa.NewManager(container, waLogger)

	pairUC := usecase.NewPairCodeUsecase(waManager)
	listUC := usecase.NewListClientsUsecase(waManager)

	handler := http.NewHandler(pairUC, listUC)
	router := http.NewRouter(handler)

	log.Printf("HTTP listening on :%s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
