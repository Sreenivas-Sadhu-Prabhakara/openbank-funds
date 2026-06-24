//go:build integration

package funds

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/sreeni/openbank-bian/pkg/obie"
	"github.com/sreeni/openbank-bian/pkg/pg"
	"github.com/sreeni/openbank-bian/pkg/testutil"
)

// newPgRepo spins up a throwaway Postgres, applies migrations and returns a
// Postgres-backed repository. Migrations are read from the module's migrations
// directory relative to this test package.
func newPgRepo(t *testing.T) *PgRepository {
	t.Helper()
	ctx := context.Background()
	dsn := testutil.PostgresDSN(t)

	pool, err := pg.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pg.RunMigrations(ctx, pool, os.DirFS("../.."), "migrations", "funds"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return NewPgRepository(pool)
}

func TestPgRepositoryFundsConfirmationRoundTrip(t *testing.T) {
	ctx := context.Background()
	repo := newPgRepo(t)

	now := time.Now().UTC().Truncate(time.Second)
	fc := &FundsConfirmation{
		FundsConfirmationID:    "fc-1",
		ConsentID:              "cons-1",
		CreationDateTime:       now,
		Reference:              "REF-9",
		InstructedAmount:       obie.MustAmount("20.00", "GBP"),
		FundsAvailable:         true,
		FundsAvailableDateTime: now,
	}
	if err := repo.Create(ctx, fc); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.Get(ctx, fc.FundsConfirmationID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ConsentID != "cons-1" || got.Reference != "REF-9" {
		t.Fatalf("unexpected confirmation %+v", got)
	}
	if !got.FundsAvailable {
		t.Fatalf("FundsAvailable = false, want true")
	}
	if got.InstructedAmount.String() != "20" || got.InstructedAmount.Currency != "GBP" {
		t.Fatalf("amount = %s %s", got.InstructedAmount, got.InstructedAmount.Currency)
	}
}

func TestPgRepositoryNegativeResultRoundTrip(t *testing.T) {
	ctx := context.Background()
	repo := newPgRepo(t)

	now := time.Now().UTC().Truncate(time.Second)
	fc := &FundsConfirmation{
		FundsConfirmationID:    "fc-2",
		ConsentID:              "cons-2",
		CreationDateTime:       now,
		InstructedAmount:       obie.MustAmount("1000000.00", "GBP"),
		FundsAvailable:         false,
		FundsAvailableDateTime: now,
	}
	if err := repo.Create(ctx, fc); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := repo.Get(ctx, fc.FundsConfirmationID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.FundsAvailable {
		t.Fatalf("FundsAvailable = true, want false")
	}
	if got.Reference != "" {
		t.Fatalf("reference = %q, want empty", got.Reference)
	}
}

func TestPgRepositoryGetMissing(t *testing.T) {
	repo := newPgRepo(t)
	if _, err := repo.Get(context.Background(), "nope"); err != ErrNotFound {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}
