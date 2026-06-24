package funds

import (
	"context"
	"sync"
)

// MemRepository is an in-memory Repository used by unit and handler tests and
// for running the service without a database. It is safe for concurrent use.
type MemRepository struct {
	mu    sync.RWMutex
	store map[string]FundsConfirmation
}

// NewMemRepository returns an empty in-memory repository.
func NewMemRepository() *MemRepository {
	return &MemRepository{store: make(map[string]FundsConfirmation)}
}

func (r *MemRepository) Create(_ context.Context, fc *FundsConfirmation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.store[fc.FundsConfirmationID] = *fc // store a copy so external mutation cannot corrupt state
	return nil
}

func (r *MemRepository) Get(_ context.Context, id string) (*FundsConfirmation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	fc, ok := r.store[id]
	if !ok {
		return nil, ErrNotFound
	}
	out := fc
	return &out, nil
}
