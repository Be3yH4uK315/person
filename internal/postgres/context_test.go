package postgres

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunner_ReturnsErrorWithoutSession(t *testing.T) {
	_, err := Runner(context.Background())

	require.Error(t, err)
}

func TestTransactionRunner_RejectsMissingTransaction(t *testing.T) {
	_, err := TransactionRunner(context.Background())
	require.Error(t, err)
}
