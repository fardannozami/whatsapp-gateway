package usecase

import (
	"context"
	"fmt"

	"github.com/fardannozami/whatsapp-gateway/internal/domain/phone"
	"github.com/fardannozami/whatsapp-gateway/internal/infra/wa"
)

type PairCodeInput struct {
	Phone   string
	Session string
}

type PairCodeOutput struct {
	Status      string // "paired_code_issued" | "already_logged_in"
	PairingCode string
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

	// client, _, err := u.wa.CreateOrGetClientByPhone(p)
	client, err := u.wa.CreateOrGetClientBySession(in.Session)
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}

	// Connect first
	if err := u.wa.EnsureConnected(ctx, in.Session, client); err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	// If already logged in
	if client.Store.ID != nil {
		_ = u.wa.RegisterClientInMemory(client)
		return &PairCodeOutput{
			Status: "already_logged_in",
		}, nil
	}

	code, err := u.wa.PairPhone(ctx, client, p)
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
