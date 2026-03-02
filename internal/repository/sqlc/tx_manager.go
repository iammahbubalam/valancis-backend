package sqlcrepo

import (
	"context"
	"valancis-backend/db/sqlc"
	"valancis-backend/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TransactionManager implements domain.TransactionManager using pgx
type TransactionManager struct {
	db *pgxpool.Pool
}

func NewTransactionManager(db *pgxpool.Pool) domain.TransactionManager {
	return &TransactionManager{db: db}
}

func (tm *TransactionManager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := tm.db.Begin(ctx)
	if err != nil {
		return err
	}

	// Create a new context with the transaction
	txCtx := context.WithValue(ctx, txKey{}, tx)

	if err := fn(txCtx); err != nil {
		tx.Rollback(ctx)
		return err
	}

	return tx.Commit(ctx)
}

type txKey struct{}

// Helper to get queries with transaction if available
func GetQueriesFromContext(ctx context.Context, defaultQueries *sqlc.Queries) *sqlc.Queries {
	if tx, ok := ctx.Value(txKey{}).(interface{ sqlc.DBTX }); ok {
		return sqlc.New(tx)
	}
	return defaultQueries
}
