package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/IlyaMudritskiy/go-big-payment/services/ledger/internal/ledger"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) CreateAccount(ctx context.Context, currency string, allowNegative bool) (ledger.Account, error) {
	var a ledger.Account

	err := s.pool.QueryRow(ctx, `
		INSERT INTO accounts (currency, allow_negative)
		VALUES ($1, $2)
		RETURNING id, currency, balance, allow_negative, created_at`,
		currency, allowNegative,
	).Scan(&a.ID, &a.Currency, &a.Balance, &a.AllowNegative, &a.CreatedAt)

	if err != nil {
		return ledger.Account{}, fmt.Errorf("insert account: %w", err)
	}
	return a, nil
}

func (s *Store) GetAccount(ctx context.Context, id uuid.UUID) (ledger.Account, error) {
	var a ledger.Account
	err := s.pool.QueryRow(ctx, `
		SELECT id, currency, balance, allow_negative, created_at
		FROM accounts WHERE id = $1`, id,
	).Scan(&a.ID, &a.Currency, &a.Balance, &a.AllowNegative, &a.CreatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return ledger.Account{}, ledger.ErrAccountNotFound
	}
	if err != nil {
		return ledger.Account{}, fmt.Errorf("select account: %w", err)
	}
	return a, nil
}

func (s *Store) Transfer(ctx context.Context, req ledger.TransferRequest) (ledger.TransferResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ledger.TransferResult{}, fmt.Errorf("begin tx: %w", err)
	}

	// В случае неудачи происходит откат транзакции
	defer func() { _ = tx.Rollback(ctx) }()

	var txID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO transactions (idempotency_key, description)
		VALUES ($1, $2)
		ON CONFLICT (idempotency_key) DO NOTHING
		RETURNING id`,
		req.IdempotencyKey, req.Description,
	).Scan(&txID)

	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `
			SELECT id FROM transactions WHERE idempotency_key = $1`, req.IdempotencyKey).Scan(&txID)
		if err != nil {
			return ledger.TransferResult{}, fmt.Errorf("select existing transaction: %w", err)
		}
		return ledger.TransferResult{TransactionID: txID, Replayed: true}, nil
	}
	if err != nil {
		return ledger.TransferResult{}, fmt.Errorf("lock accounts: %w", err)
	}

	// Блокировка обоих счетов в порядке id для защиты от дедлоков
	rows, err := tx.Query(ctx, `
		SELECT id, currency, balance, allow_negative, created_at
		FROM accounts
		WHERE id IN ($1, $2)
		ORDER BY id
		FOR UPDATE`, req.FromAccountID, req.ToAccountID)

	if err != nil {
		return ledger.TransferResult{}, fmt.Errorf("lock accounts: %w", err)
	}

	accounts, err := pgx.CollectRows(rows, pgx.RowToStructByPos[ledger.Account])
	if err != nil {
		return ledger.TransferResult{}, fmt.Errorf("scan accounts: %w", err)
	}

	byID := make(map[uuid.UUID]ledger.Account, len(accounts))
	for _, account := range accounts {
		byID[account.ID] = account
	}

	from, okFrom := byID[req.FromAccountID]
	to, okTo := byID[req.ToAccountID]

	if !okFrom || !okTo {
		return ledger.TransferResult{}, ledger.ErrAccountNotFound
	}

	if err := ledger.CheckTransfer(from, to, req.Amount, req.Currency); err != nil {
		return ledger.TransferResult{}, err
	}

	// Проводки
	_, err = tx.Exec(ctx, `
		INSERT INTO entries (transaction_id, account_id, amount)
		VALUES ($1, $2, $3), ($1, $4, $5)`,
		txID, from.ID, -req.Amount, to.ID, req.Amount)

	if err != nil {
		return ledger.TransferResult{}, fmt.Errorf("insert entries: %w", err)
	}

	for _, entrs := range []struct {
		id     uuid.UUID
		amount int64
	}{
		{from.ID, -req.Amount},
		{to.ID, req.Amount},
	} {
		if _, err := tx.Exec(ctx,
			`UPDATE accounts SET balance = balance + $2 where id = $1`, entrs.id, entrs.amount,
		); err != nil {
			return ledger.TransferResult{}, fmt.Errorf("update balance: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return ledger.TransferResult{}, fmt.Errorf("commit: %w", err)
	}
	return ledger.TransferResult{TransactionID: txID}, nil
}
