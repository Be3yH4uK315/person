package postgres

import (
	"context"
	"fmt"

	"github.com/gocraft/dbr/v2"
)

type sessionContextKey struct{}
type transactionContextKey struct{}

func WithSession(ctx context.Context, session *dbr.Session) context.Context {
	return context.WithValue(ctx, sessionContextKey{}, session)
}

func WithTransaction(ctx context.Context, tx *dbr.Tx) context.Context {
	return context.WithValue(ctx, transactionContextKey{}, tx)
}

func sessionFromContext(ctx context.Context) (*dbr.Session, error) {
	session, ok := ctx.Value(sessionContextKey{}).(*dbr.Session)
	if !ok || session == nil {
		return nil, fmt.Errorf("dbr session is missing from context")
	}
	return session, nil
}

func Runner(ctx context.Context) (dbr.SessionRunner, error) {
	if tx, ok := ctx.Value(transactionContextKey{}).(*dbr.Tx); ok && tx != nil {
		return tx, nil
	}
	return sessionFromContext(ctx)
}

func TransactionRunner(ctx context.Context) (dbr.SessionRunner, error) {
	tx, ok := ctx.Value(transactionContextKey{}).(*dbr.Tx)
	if !ok || tx == nil {
		return nil, fmt.Errorf("dbr transaction is missing from context")
	}
	return tx, nil
}
