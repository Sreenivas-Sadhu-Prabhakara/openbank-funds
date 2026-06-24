package funds

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sreeni/openbank-bian/pkg/obie"
)

// PgRepository is the Postgres-backed Repository. The funds service owns the
// "funds" schema; this type touches nothing outside it.
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewPgRepository returns a Postgres repository over the given pool.
func NewPgRepository(pool *pgxpool.Pool) *PgRepository {
	return &PgRepository{pool: pool}
}

const fundsColumns = `id, consent_id, creation_dt, reference,
	instructed_amount, instructed_currency,
	funds_available, funds_available_dt`

func (r *PgRepository) Create(ctx context.Context, fc *FundsConfirmation) error {
	amount := fc.InstructedAmount.String()
	currency := fc.InstructedAmount.Currency

	_, err := r.pool.Exec(ctx, `
		INSERT INTO funds.funds_confirmations (`+fundsColumns+`)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		fc.FundsConfirmationID, fc.ConsentID, fc.CreationDateTime, nullable(fc.Reference),
		amount, currency, fc.FundsAvailable, fc.FundsAvailableDateTime,
	)
	return err
}

func (r *PgRepository) Get(ctx context.Context, id string) (*FundsConfirmation, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+fundsColumns+` FROM funds.funds_confirmations WHERE id = $1`, id)
	fc, err := scanFundsConfirmation(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return fc, err
}

// scanFundsConfirmation reads a row in fundsColumns order into a FundsConfirmation.
func scanFundsConfirmation(row pgx.Row) (*FundsConfirmation, error) {
	var (
		fc               FundsConfirmation
		reference        *string
		amount, currency string
	)
	if err := row.Scan(
		&fc.FundsConfirmationID, &fc.ConsentID, &fc.CreationDateTime, &reference,
		&amount, &currency,
		&fc.FundsAvailable, &fc.FundsAvailableDateTime,
	); err != nil {
		return nil, err
	}

	fc.Reference = deref(reference)
	amt, err := obie.NewAmount(amount, currency)
	if err != nil {
		return nil, err
	}
	fc.InstructedAmount = amt
	return &fc, nil
}

func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
