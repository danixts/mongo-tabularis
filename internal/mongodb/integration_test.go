package mongodb

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestIntegrationConnection(t *testing.T) {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		t.Skip("set MONGODB_URI to run integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool := NewPool()
	defer pool.Close()

	client, database, err := pool.Acquire(ctx, ConnParams{ConnectionString: uri})
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}

	if err := Ping(ctx, client, database); err != nil {
		t.Fatalf("ping: %v", err)
	}

	if databases := ListDatabases(ctx, client, database); len(databases.([]string)) == 0 {
		t.Error("expected at least one database")
	}
}
