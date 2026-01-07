package usecase

import (
	"context"
	"fmt"

	"github.com/fardannozami/whatsapp-gateway/internal/infra/wa"
	"go.mau.fi/whatsmeow/types"
)

type MeOutput struct {
	Status   string
	ID       string
	JID      string
	PushName string
}

type MeUsecase struct {
	wa *wa.Manager
}

func NewMeUsecase(waManager *wa.Manager) *MeUsecase {
	return &MeUsecase{wa: waManager}
}

func (u *MeUsecase) Execute(ctx context.Context, session string) (*MeOutput, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	client, err := u.wa.CreateOrGetClientBySession(session)
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}

	id := ""
	if client.Store.ID != nil {
		id = client.Store.ID.User
	}

	out := &MeOutput{
		Status:   u.wa.SessionStatus(session, client),
		ID:       id,
		JID:      jidString(client.Store.ID),
		PushName: client.Store.PushName,
	}

	return out, nil
}

func jidString(jid *types.JID) string {
	if jid == nil {
		return ""
	}
	return types.JID(*jid).String()
}
