package ledger

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Ошибки
var (
	ErrInvalidArgument   = errors.New("invalid argument")
	ErrAccountNotFound   = errors.New("account not found")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrCurrencyMismatch  = errors.New("currency mismatch")
)

type Account struct {
	ID            uuid.UUID
	Currency      string
	Balance       int64
	AllowNegative bool
	CreatedAt     time.Time
}

type TransferRequest struct {
	IdempotencyKey string
	FromAccountID  uuid.UUID
	ToAccountID    uuid.UUID
	Amount         int64
	Currency       string
	Description    string
}

type TransferResult struct {
	TransactionID uuid.UUID
	Replayed      bool // повтор выполненного перевода, денежные средства повторно не перемещались
}

func (r TransferRequest) Validate() error {
	switch {
	case r.IdempotencyKey == "":
		return fmt.Errorf("%w: idempotancy key is required", ErrInvalidArgument)
	case r.Amount <= 0:
		return fmt.Errorf("%w: amount must be positive", ErrInvalidArgument)
	case r.FromAccountID == r.ToAccountID:
		return fmt.Errorf("%w: cannot transfer to the same account", ErrInvalidArgument)
	case r.Currency == "":
		return fmt.Errorf("%w: currency is required", ErrInvalidArgument)
	}
	return nil
}

// Проверка возможности перевода при текущем состоянии счетов
func CheckTransfer(from Account, to Account, amount int64, currency string) error {
	if from.Currency != to.Currency || to.Currency != currency {
		return fmt.Errorf("%w: accounts currency %s/%s, transfer currency [%s]", ErrCurrencyMismatch, from.Currency, to.Currency, currency)
	}
	if !from.AllowNegative && from.Balance < amount {
		return ErrInsufficientFunds
	}
	return nil
}
