package cache

import (
	"testing"
	"time"
)

func TestNewLocalCache(t *testing.T) {

	cache, err := NewLocalCache(time.Second * 1)

	if err != nil {
		t.Error(err)
	}

	err = cache.Cache.Set("test-key", []byte("test-data"))

	if err != nil {
		t.Error(err)
	}

	data, err := cache.Cache.Get("test-key")

	if err != nil {
		t.Error(err)
	}

	if string(data) != "test-data" {
		t.Error("data not match")
	}

}
