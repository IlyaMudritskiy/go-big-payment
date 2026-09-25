package ledger

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestCheckTransfer(t *testing.T) {
	tests := []struct {
		name    string
		from    Account
		amount  int64
		wantErr error
	}{
		{"enough funds", Account{Currency: "RUB", Balance: 100}, 100, nil},
		{"insufficient funds", Account{Currency: "RUB", Balance: 99}, 100, ErrInsufficientFunds},
		{"system account may go negative", Account{Currency: "RUB", AllowNegative: true}, 100, nil},
		{"currency mismatch", Account{Currency: "USD", Balance: 1000}, 100, ErrCurrencyMismatch},
	}

	to := Account{Currency: "RUB"}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckTransfer(tt.from, to, tt.amount, "RUB")
			if !errors.Is(err, tt.wantErr) {
				t.Error("CheckTransfer() error = %w, want %w", err, tt.wantErr)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	uuidFrom := uuid.New()
	uuidTo := uuid.New()

	tests := []struct {
		name    string
		tr      TransferRequest
		wantErr error
	}{
		{
			"no idempotency key",
			TransferRequest{
				IdempotencyKey: "",
				FromAccountID:  uuidFrom,
				ToAccountID:    uuidTo,
				Amount:         10000,
				Currency:       "RUB",
				Description:    "",
			},
			ErrInvalidArgument,
		},
		{
			"zero amount",
			TransferRequest{
				IdempotencyKey: "abcdefg",
				FromAccountID:  uuidFrom,
				ToAccountID:    uuidTo,
				Amount:         0,
				Currency:       "RUB",
				Description:    "",
			},
			ErrInvalidArgument,
		},
		{
			"negative amount",
			TransferRequest{
				IdempotencyKey: "asdeefn",
				FromAccountID:  uuidFrom,
				ToAccountID:    uuidTo,
				Amount:         -10000,
				Currency:       "RUB",
				Description:    "",
			},
			ErrInvalidArgument,
		},
		{
			"same account",
			TransferRequest{
				IdempotencyKey: "nfhsele",
				FromAccountID:  uuidFrom,
				ToAccountID:    uuidFrom,
				Amount:         10000,
				Currency:       "RUB",
				Description:    "",
			},
			ErrInvalidArgument,
		},
		{
			"incorrect currency",
			TransferRequest{
				IdempotencyKey: "ffhsdlk",
				FromAccountID:  uuidFrom,
				ToAccountID:    uuidTo,
				Amount:         10000,
				Currency:       "1xB",
				Description:    "",
			},
			ErrInvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tr.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Error("TransferRequest.Validate() error = %w, want %w", err, tt.wantErr)
			}
		})
	}
}
