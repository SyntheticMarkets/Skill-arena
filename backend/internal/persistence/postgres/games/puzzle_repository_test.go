package games

import (
	"errors"
	"testing"

	"github.com/lib/pq"
)

func TestRetryableTransactionErrorOnlyAcceptsTransientPostgresFailures(t *testing.T) {
	for _, code := range []pq.ErrorCode{"40001", "40P01"} {
		if !retryableTransactionError(&pq.Error{Code: code}) {
			t.Fatalf("PostgreSQL code %s should be retried", code)
		}
	}
	if retryableTransactionError(&pq.Error{Code: "23505"}) {
		t.Fatal("uniqueness violations must not be retried")
	}
	if retryableTransactionError(errors.New("network unavailable")) {
		t.Fatal("unclassified errors must not be retried")
	}
}
