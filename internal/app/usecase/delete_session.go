package usecase

import (
	"context"

	"github.com/fardannozami/whatsapp-gateway/internal/infra/wa"
)

type DeleteSessionUsecase struct {
	wa *wa.Manager
}

func NewDeleteSessionUsecase(waManager *wa.Manager) *DeleteSessionUsecase {
	return &DeleteSessionUsecase{wa: waManager}
}

func (u *DeleteSessionUsecase) Execute(ctx context.Context, session string) (bool, error) {
	return u.wa.DeleteSession(ctx, session)
}
