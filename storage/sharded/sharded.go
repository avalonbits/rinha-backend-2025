package sharded

import (
	"context"
	"errors"
	"fmt"
	"hash/maphash"
	"sync"

	"github.com/avalonbits/rinha-backend-2025/storage"
	"github.com/avalonbits/rinha-backend-2025/storage/datastore"
)

type Store struct {
	dbs  []*storage.DB[datastore.Queries]
	seed maphash.Seed
}

func New(shards int, database string) *Store {
	if shards < 1 {
		shards = 1
	}

	dbs := make([]*storage.DB[datastore.Queries], 0, shards)
	for i := range shards {
		db, err := storage.GetDB(fmt.Sprintf("%s-%d", database, i+1), datastore.Migrations, datastore.Factory)
		if err != nil {
			panic(err)
		}
		dbs = append(dbs, db)
	}

	return &Store{
		dbs:  dbs,
		seed: maphash.MakeSeed(),
	}
}

func (s *Store) Close() {
	for _, db := range s.dbs {
		db.Close()
	}
}

func (s *Store) ShardCount() int {
	return len(s.dbs)
}

func (s *Store) Write(ctx context.Context, key string, f func(queries *datastore.Queries) error) error {
	return s.getDB(key).Write(ctx, f)
}

func (s *Store) Read(ctx context.Context, key string, f func(queries *datastore.Queries) error) error {
	return s.getDB(key).Read(ctx, f)
}

func (s *Store) ReadAll(ctx context.Context, f func(shard int, queries *datastore.Queries) error) error {
	var wg sync.WaitGroup
	wg.Add(len(s.dbs))

	errs := make([]error, len(s.dbs))
	for i, db := range s.dbs {
		go func() {
			defer wg.Done()
			errs[i] = db.Read(ctx, func(queries *datastore.Queries) error {
				return f(i, queries)
			})
		}()
	}
	wg.Wait()

	return errors.Join(errs...)
}

func (s *Store) getDB(key string) *storage.DB[datastore.Queries] {
	var h maphash.Hash
	h.SetSeed(s.seed)
	h.WriteString(key)

	idx := h.Sum64() % uint64(len(s.dbs))
	return s.dbs[idx]
}
