package funds

import (
	"context"
	"errors"
)

// ErrNotFound is returned by a Repository when no funds confirmation matches the
// id.
var ErrNotFound = errors.New("funds confirmation not found")

// Repository is the persistence port for funds confirmations. Both the in-memory
// and the Postgres implementations satisfy it, and the service layer depends only
// on this interface — so the same business-logic tests run against either store.
//
// CBPII has no public GET endpoint; Get exists so the audit record can be read
// back (for example by the integration tests that round-trip a confirmation).
type Repository interface {
	Create(ctx context.Context, fc *FundsConfirmation) error
	Get(ctx context.Context, id string) (*FundsConfirmation, error)
}
