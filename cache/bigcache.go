package cache

import (
	"context"
	"github.com/allegro/bigcache/v3"
	"time"
)

type BigCache struct {
	Cache *bigcache.BigCache
}

func NewBigCache(allKeysExpTime time.Duration) (*BigCache, error) {

	cache, err := bigcache.New(context.Background(), bigcache.DefaultConfig(allKeysExpTime))

	if err != nil {
		return nil, err
	}
	return &BigCache{Cache: cache}, nil
}

func (s *BigCache) Set(key string, entry []byte) (err error) {
	return s.Cache.Set(key, entry)
}

func (s *BigCache) Get(key string) ([]byte, error) {
	return s.Cache.Get(key)
}
