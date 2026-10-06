package readers

import (
	"context"
	"testing"

	"person/internal/logic/ports"
)

func TestEscapeLikePattern(t *testing.T) {
	got := escapeLikePattern(`100%_\users`)
	if got != `100\%\_\\users` {
		t.Fatalf("unexpected escaped pattern: %q", got)
	}
}

func TestPersonReaderListRejectsNegativePaginationBeforeConversion(t *testing.T) {
	reader := NewPersonReader()
	for _, query := range []ports.ListPersonsQuery{
		{Limit: -1, Offset: 0},
		{Limit: 20, Offset: -1},
	} {
		if _, err := reader.List(context.Background(), query); err == nil {
			t.Fatalf("expected error for negative pagination: %+v", query)
		}
	}
}

func TestNonNegativeUint64(t *testing.T) {
	if got, ok := nonNegativeUint64(0); !ok || got != 0 {
		t.Fatalf("nonNegativeUint64(0) = (%d, %t), want (0, true)", got, ok)
	}
	if _, ok := nonNegativeUint64(-1); ok {
		t.Fatal("nonNegativeUint64(-1) unexpectedly succeeded")
	}
	maxInt := int(^uint(0) >> 1)
	if got, ok := nonNegativeUint64(maxInt); !ok || got != uint64(maxInt) {
		t.Fatalf("nonNegativeUint64(maxInt) = (%d, %t), want (%d, true)", got, ok, uint64(maxInt))
	}
}
