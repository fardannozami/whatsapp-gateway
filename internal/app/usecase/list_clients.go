package usecase

import "github.com/fardannozami/whatsapp-gateway/internal/infra/wa"

type ListClientsUsecase struct {
	wa *wa.Manager
}

func NewListClientsUsecase(waManager *wa.Manager) *ListClientsUsecase {
	return &ListClientsUsecase{wa: waManager}
}

func (u *ListClientsUsecase) Execute() []string {
	return u.wa.ListClients()
}
