package main

import (
	"context"
	"log"

	"github.com/fardannozami/whatsapp-gateway/internal/app/http"
	"github.com/fardannozami/whatsapp-gateway/internal/app/usecase"
	"github.com/fardannozami/whatsapp-gateway/internal/config"
	"github.com/fardannozami/whatsapp-gateway/internal/infra/wa"
	walog "go.mau.fi/whatsmeow/util/log"
)

func main() {
	cfg := config.Load()

	waLogger := walog.Stdout("WA", "ERROR", true)

	waManager := wa.NewManager(cfg.SQLitePath, waLogger)
	go func() {
		if err := waManager.AutoConnectExisting(context.Background()); err != nil {
			log.Printf("auto connect existing sessions: %v", err)
		}
	}()

	pairUC := usecase.NewPairCodeUsecase(waManager)
	listUC := usecase.NewListClientsUsecase(waManager)
	meUC := usecase.NewMeUsecase(waManager)
	pairSU := usecase.NewPairStreamUsecase(waManager)
	sessUC := usecase.NewListSessionsUsecase(waManager)
	delUC := usecase.NewDeleteSessionUsecase(waManager)
	stopUC := usecase.NewStopSessionUsecase(waManager)
	delFUC := usecase.NewDeleteSessionForceUsecase(waManager)
	sendUC := usecase.NewSendTextUsecase(waManager)

	handler := http.NewHandler(pairUC, listUC, meUC, pairSU, sessUC, delUC, stopUC, delFUC, sendUC)
	router := http.NewRouter(handler)

	log.Printf("HTTP listening on :%s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
