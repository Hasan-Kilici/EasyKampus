package cache

import (
	"context"
	"strings"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

type Memory struct {
	cache *expirable.LRU[string, any]
}

func New(size int, ttl time.Duration) (*Memory, error) {
	return &Memory{cache: expirable.NewLRU[string, any](size, nil, ttl)}, nil
}

func (memory *Memory) Get(_ context.Context, key string) (any, bool) {
	return memory.cache.Get(key)
}

func (memory *Memory) Set(_ context.Context, key string, value any) error {
	memory.cache.Add(key, value)
	return nil
}

func (memory *Memory) Delete(_ context.Context, key string) error {
	memory.cache.Remove(key)
	return nil
}

func (memory *Memory) DeletePrefix(_ context.Context, prefix string) error {
	for _, key := range memory.cache.Keys() {
		if strings.HasPrefix(key, prefix) {
			memory.cache.Remove(key)
		}
	}
	return nil
}

func (memory *Memory) Clear(_ context.Context) error {
	memory.cache.Purge()
	return nil
}
