package app

import (
	"github.com/gin-contrib/sessions"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store interface {
	sessions.Store
}

type store struct {
	*PGStore
}

// NewPGStore creates a new PGStore instance and a new pgxpool.Pool.
// This will also create in the database the schema needed by pgstore.
func NewPGStore(pool *pgxpool.Pool, keyPairs ...[]byte) (Store, error) {
	p, err := NewPGStoreFromPool(pool, keyPairs...)
	if err != nil {
		return nil, err
	}

	return &store{p}, nil
}

func (s *store) Options(options sessions.Options) {
	s.PGStore.Options = options.ToGorillaOptions()
}
