package usecase

import (
	"context"
	"fmt"

	"github.com/fardannozami/whatsapp-gateway/internal/domain/phone"
	"github.com/fardannozami/whatsapp-gateway/internal/infra/wa"
	"go.mau.fi/whatsmeow"
)

type PairCodeInput struct {
	Phone      string
	ClientType string
}

type PairCodeOutput struct {
	Status      string // "paired_code_issued" | "already_logged_in"
	PairingCode string
	JID         string
}

type PairCodeUsecase struct {
	wa *wa.Manager
}

func NewPairCodeUsecase(waManager *wa.Manager) *PairCodeUsecase {
	return &PairCodeUsecase{wa: waManager}
}

func (u *PairCodeUsecase) Execute(ctx context.Context, in PairCodeInput) (*PairCodeOutput, error) {
	p, err := phone.Normalize(in.Phone)
	if err != nil {
		return nil, err
	}

	client, _, err := u.wa.CreateOrGetClientByPhone(p)
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}

	// Connect first
	if err := u.wa.EnsureConnected(ctx, client); err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	// If already logged in
	if client.Store.ID != nil {
		jid := client.Store.ID.String()
		_ = u.wa.RegisterClientInMemory(client)
		return &PairCodeOutput{
			Status: "already_logged_in",
			JID:    jid,
		}, nil
	}

	code, err := u.wa.PairPhone(ctx, client, p, whatsmeow.PairClientChrome)
	if err != nil {
		return nil, fmt.Errorf("pair phone: %w", err)
	}

	// Setelah user input pairing code di HP, login akan terjadi.
	// JID bisa belum langsung muncul di momen ini.
	return &PairCodeOutput{
		Status:      "paired_code_issued",
		PairingCode: code,
	}, nil
}
