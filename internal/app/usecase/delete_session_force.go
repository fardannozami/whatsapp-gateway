package usecase

import (
	"context"

	"github.com/fardannozami/whatsapp-gateway/internal/infra/wa"
)

type DeleteSessionForceUsecase struct {
	wa *wa.Manager
}

func NewDeleteSessionForceUsecase(waManager *wa.Manager) *DeleteSessionForceUsecase {
	return &DeleteSessionForceUsecase{wa: waManager}
}

func (u *DeleteSessionForceUsecase) Execute(ctx context.Context, session string) (bool, error) {
	return u.wa.DeleteSessionForce(ctx, session)
}
