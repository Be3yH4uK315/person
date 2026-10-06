package postgres

import (
	"context"
	"fmt"
)

type UnitOfWork struct{}

func NewUnitOfWork() *UnitOfWork { return &UnitOfWork{} }

func (u *UnitOfWork) Do(ctx context.Context, fn func(context.Context) error) error {
	session, err := sessionFromContext(ctx)
	if err != nil {
		return err
	}
	tx, err := session.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.RollbackUnlessCommitted()

	if err := fn(WithTransaction(ctx, tx)); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}
