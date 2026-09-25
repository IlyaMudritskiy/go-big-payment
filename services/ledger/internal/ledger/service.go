package ledger

import (
	"context"

	"github.com/google/uuid"
)

type Store interface {
	CreateAccount(ctx context.Context, currency string, allowNegative bool) (Account, error)
	GetAccount(ctx context.Context, id uuid.UUID) (Account, error)
	Transfer(ctx context.Context, req TransferRequest) (TransferResult, error)
}

type Service struct {
	store Store
}

func NewSerivce(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) CreateAccount(ctx context.Context, currency string, allowNegative bool) (Account, error) {
	return s.store.CreateAccount(ctx, currency, allowNegative)
}

func (s *Service) GetAccount(ctx context.Context, id uuid.UUID) (Account, error) {
	return s.store.GetAccount(ctx, id)
}

func (s *Service) Transfer(ctx context.Context, req TransferRequest) (TransferResult, error) {
	if err := req.Validate(); err != nil {
		return TransferResult{}, err
	}
	return s.store.Transfer(ctx, req)
}
