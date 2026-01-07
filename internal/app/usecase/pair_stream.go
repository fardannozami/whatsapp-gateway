package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fardannozami/whatsapp-gateway/internal/infra/wa"
)

type PairStreamOutput struct {
	Status      string
	PairingCode string
	ExpiresIn   int
	RetryIn     int
	Detail      string
}

type PairStreamUsecase struct {
	wa *wa.Manager
}

func NewPairStreamUsecase(waManager *wa.Manager) *PairStreamUsecase {
	return &PairStreamUsecase{wa: waManager}
}

func (u *PairStreamUsecase) Next(ctx context.Context, session string) (*PairStreamOutput, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, true, err
	}

	client, err := u.wa.CreateOrGetClientBySession(session)
	if err != nil {
		return nil, false, fmt.Errorf("create client: %w", err)
	}

	status := u.wa.SessionStatus(session, client)
	if status != "working" && status != "connecting" {
		if err := u.wa.EnsureConnected(ctx, session, client); err != nil {
			return &PairStreamOutput{Status: "failed"}, false, err
		}
	}

	if client.Store.ID != nil {
		_ = u.wa.RegisterClientInMemory(client)
		_ = u.wa.ClearPairing(session)
		return &PairStreamOutput{Status: "working"}, true, nil
	}

	state, ok := u.wa.GetPairingState(session)
	if !ok || state.Phone == "" {
		return &PairStreamOutput{Status: "need_phone"}, false, nil
	}

	now := time.Now()
	if !state.NextRetryAt.IsZero() && state.NextRetryAt.After(now) && state.Code == "" {
		retryIn := int(state.NextRetryAt.Sub(now).Seconds())
		if retryIn < 0 {
			retryIn = 0
		}
		status := "failed"
		if backoff, s := pairingBackoff(errors.New(state.LastError)); backoff > 0 {
			status = s
		}
		return &PairStreamOutput{
			Status:  status,
			RetryIn: retryIn,
			Detail:  state.LastError,
		}, false, nil
	}

	if state.Code != "" && state.ExpiresAt.After(now) {
		expiresIn := int(state.ExpiresAt.Sub(now).Seconds())
		if expiresIn < 0 {
			expiresIn = 0
		}
		return &PairStreamOutput{
			Status:      "waiting",
			PairingCode: state.Code,
			ExpiresIn:   expiresIn,
		}, false, nil
	}

	code, err := u.wa.PairPhone(ctx, client, state.Phone)
	if err != nil {
		backoff, status := pairingBackoff(err)
		_ = u.wa.UpdatePairingFailure(session, err.Error(), now, backoff)
		return &PairStreamOutput{
			Status:  status,
			RetryIn: int(backoff.Seconds()),
			Detail:  err.Error(),
		}, false, nil
	}
	if err := u.wa.UpdatePairingCode(session, code, now, pairCodeTTL); err != nil {
		return nil, false, fmt.Errorf("update pairing code: %w", err)
	}

	return &PairStreamOutput{
		Status:      "waiting",
		PairingCode: code,
		ExpiresIn:   int(pairCodeTTL.Seconds()),
	}, false, nil
}
