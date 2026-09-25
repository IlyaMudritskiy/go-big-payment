package postgres

import (
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/IlyaMudritskiy/go-big-payment/services/ledger/internal/ledger"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	pool, err := pgxpool.New(t.Context(), url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return NewStore(pool)
}

// Подготовка данных
func mustTransfer(t *testing.T, s *Store, from uuid.UUID, to uuid.UUID, amount int64) {
	t.Helper()
	_, err := s.Transfer(t.Context(), ledger.TransferRequest{
		IdempotencyKey: uuid.NewString(),
		FromAccountID:  from,
		ToAccountID:    to,
		Amount:         amount,
		Currency:       "RUB",
	})
	if err != nil {
		t.Fatalf("transfer: %v", err)
	}
}

func mustCreateAcoount(t *testing.T, s *Store, allowNegative bool) ledger.Account {
	t.Helper()
	a, err := s.CreateAccount(t.Context(), "RUB", allowNegative)
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	return a
}

func TestTransfer_Concurrent(t *testing.T) {
	s := newTestStore(t)
	ctx := t.Context()

	world := mustCreateAcoount(t, s, true)
	a := mustCreateAcoount(t, s, false)
	b := mustCreateAcoount(t, s, false)
	mustTransfer(t, s, world.ID, a.ID, 1000)

	const n = 100
	var wg sync.WaitGroup
	errs := make(chan error, n)

	for i := range n {
		from, to := a.ID, b.ID
		if i%2 == 0 {
			from, to = b.ID, a.ID // Делаем половину всех переводов обратно, тестируем на дедлоки
		}
		wg.Go(func() {
			_, err := s.Transfer(ctx, ledger.TransferRequest{
				IdempotencyKey: uuid.NewString(),
				FromAccountID:  from,
				ToAccountID:    to,
				Amount:         10,
				Currency:       "RUB",
			})
			if err != nil && !errors.Is(err, ledger.ErrInsufficientFunds) {
				errs <- err
			}
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("unexpected transfer error: %v", err)
	}

	gotA, err := s.GetAccount(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	gotB, err := s.GetAccount(ctx, b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if sum := gotA.Balance + gotB.Balance; sum != 1000 {
		t.Errorf("A+B = %d, want 1000 — деньги появились или исчезли", sum)
	}
	if gotA.Balance < 0 || gotB.Balance < 0 {
		t.Errorf("negative balance: A=%d B=%d", gotA.Balance, gotB.Balance)
	}
}

func TestTransfer_Idempotent(t *testing.T) {
	s := newTestStore(t)
	ctx := t.Context()

	world := mustCreateAcoount(t, s, true)
	to := mustCreateAcoount(t, s, false)

	req := ledger.TransferRequest{
		IdempotencyKey: uuid.NewString(),
		FromAccountID:  world.ID,
		ToAccountID:    to.ID,
		Amount:         100,
		Currency:       "RUB",
	}

	const n = 10

	type outcome struct {
		res ledger.TransferResult
		err error
	}
	outcomes := make(chan outcome, n)

	start := make(chan struct{})

	var wg sync.WaitGroup
	for range n {
		wg.Go(func() {
			<-start
			res, err := s.Transfer(ctx, req)
			outcomes <- outcome{res: res, err: err}
		})
	}
	close(start)
	wg.Wait()
	close(outcomes)

	var executed int
	txIDs := make(map[uuid.UUID]struct{})

	for o := range outcomes {
		if o.err != nil {
			t.Errorf("unexpected transfer error: %v", o.err)
			continue
		}
		if !o.res.Replayed {
			executed++
		}
		txIDs[o.res.TransactionID] = struct{}{}
	}

	if executed != 1 {
		t.Errorf("executed %d times, want exactly 1", executed)
	}
	if len(txIDs) != 1 {
		t.Errorf("got %d different transaction IDs, want 1", len(txIDs))
	}

	got, err := s.GetAccount(ctx, to.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Balance != req.Amount {
		t.Errorf("balance = %d, want %d — деньги зачислены больше одного раза", got.Balance, req.Amount)
	}
}

