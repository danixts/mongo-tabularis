package mongodb

import (
	"fmt"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

const columnsCacheTTL = 30 * time.Second

type cachedColumns struct {
	columns []Column
	expires time.Time
}

type columnStore struct {
	mu      sync.RWMutex
	entries map[string]cachedColumns
	ttl     time.Duration
}

func newColumnStore(ttl time.Duration) *columnStore {
	return &columnStore{entries: make(map[string]cachedColumns), ttl: ttl}
}

func (s *columnStore) load(key string) ([]Column, bool) {
	s.mu.RLock()
	entry, ok := s.entries[key]
	s.mu.RUnlock()
	if !ok || time.Now().After(entry.expires) {
		return nil, false
	}
	return entry.columns, true
}

func (s *columnStore) store(key string, columns []Column) {
	s.mu.Lock()
	s.entries[key] = cachedColumns{columns: columns, expires: time.Now().Add(s.ttl)}
	s.mu.Unlock()
}

func (s *columnStore) invalidate(key string) {
	s.mu.Lock()
	delete(s.entries, key)
	s.mu.Unlock()
}

var columnsCache = newColumnStore(columnsCacheTTL)

func columnsKey(client *mongo.Client, database, collection string) string {
	return fmt.Sprintf("%p|%s|%s", client, database, collection)
}
