package usecase

import (
	"context"

	"github.com/fardannozami/whatsapp-gateway/internal/infra/wa"
)

type SessionItem struct {
	Session  string
	ID       string
	PushName string
	Status   string
}

type ListSessionsUsecase struct {
	wa *wa.Manager
}

func NewListSessionsUsecase(waManager *wa.Manager) *ListSessionsUsecase {
	return &ListSessionsUsecase{wa: waManager}
}

func (u *ListSessionsUsecase) Execute(ctx context.Context) ([]SessionItem, error) {
	infos, err := u.wa.ListSessions(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]SessionItem, 0, len(infos))
	for _, info := range infos {
		out = append(out, SessionItem{
			Session:  info.Session,
			ID:       info.ID,
			PushName: info.PushName,
			Status:   info.Status,
		})
	}

	return out, nil
}
