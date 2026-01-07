package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/fardannozami/whatsapp-gateway/internal/domain/phone"
	"github.com/fardannozami/whatsapp-gateway/internal/infra/wa"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

type SendTextInput struct {
	Session string
	To      string
	Message string
}

type SendTextOutput struct {
	Status    string
	MessageID string
}

type SendTextUsecase struct {
	wa *wa.Manager
}

func NewSendTextUsecase(waManager *wa.Manager) *SendTextUsecase {
	return &SendTextUsecase{wa: waManager}
}

func (u *SendTextUsecase) Execute(ctx context.Context, in SendTextInput) (*SendTextOutput, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if strings.TrimSpace(in.Session) == "" {
		return nil, fmt.Errorf("session is required")
	}
	if strings.TrimSpace(in.To) == "" {
		return nil, fmt.Errorf("to is required")
	}
	if strings.TrimSpace(in.Message) == "" {
		return nil, fmt.Errorf("message is required")
	}

	client, err := u.wa.CreateOrGetClientBySession(in.Session)
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}

	if err := u.wa.EnsureConnected(ctx, in.Session, client); err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	if client.Store.ID == nil {
		return nil, fmt.Errorf("session not logged in")
	}

	recipient, err := parseRecipient(in.To)
	if err != nil {
		return nil, err
	}

	msg := &waE2E.Message{Conversation: proto.String(in.Message)}
	resp, err := client.SendMessage(ctx, recipient, msg)
	if err != nil {
		return nil, err
	}

	out := &SendTextOutput{Status: "sent"}
	if resp.ID != "" {
		out.MessageID = resp.ID
	}
	return out, nil
}

func parseRecipient(raw string) (types.JID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return types.JID{}, fmt.Errorf("to is required")
	}

	if strings.Contains(raw, "@") {
		jid, err := types.ParseJID(raw)
		if err != nil {
			return types.JID{}, err
		}
		return jid, nil
	}

	normalized, err := phone.Normalize(raw)
	if err != nil {
		return types.JID{}, err
	}

	return types.JID{User: normalized, Server: "s.whatsapp.net"}, nil
}
