package http

import (
	"github.com/fardannozami/whatsapp-gateway/internal/app/usecase"
)

type Handler struct {
	pairUC *usecase.PairCodeUsecase
	listUC *usecase.ListClientsUsecase
	meUC   *usecase.MeUsecase
	pairSU *usecase.PairStreamUsecase
	sessUC *usecase.ListSessionsUsecase
	delUC  *usecase.DeleteSessionUsecase
	stopUC *usecase.StopSessionUsecase
	delFUC *usecase.DeleteSessionForceUsecase
	sendUC *usecase.SendTextUsecase
}

func NewHandler(pairUC *usecase.PairCodeUsecase, listUC *usecase.ListClientsUsecase, meUC *usecase.MeUsecase, pairSU *usecase.PairStreamUsecase, sessUC *usecase.ListSessionsUsecase, delUC *usecase.DeleteSessionUsecase, stopUC *usecase.StopSessionUsecase, delFUC *usecase.DeleteSessionForceUsecase, sendUC *usecase.SendTextUsecase) *Handler {
	return &Handler{
		pairUC: pairUC,
		listUC: listUC,
		meUC:   meUC,
		pairSU: pairSU,
		sessUC: sessUC,
		delUC:  delUC,
		stopUC: stopUC,
		delFUC: delFUC,
		sendUC: sendUC,
	}
}
