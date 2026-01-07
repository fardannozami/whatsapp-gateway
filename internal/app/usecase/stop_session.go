package usecase

import (
	"context"

	"github.com/fardannozami/whatsapp-gateway/internal/infra/wa"
)

type StopSessionUsecase struct {
	wa *wa.Manager
}

func NewStopSessionUsecase(waManager *wa.Manager) *StopSessionUsecase {
	return &StopSessionUsecase{wa: waManager}
}

func (u *StopSessionUsecase) Execute(ctx context.Context, session string) (bool, error) {
	return u.wa.StopSession(ctx, session)
}
