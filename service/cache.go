package service

import (
	"context"

	"github.com/dimcz/tgpollbot/storage"
	"github.com/go-redis/redis/v8"
	"github.com/jellydator/ttlcache/v3"
)

type Cache struct {
	rc    *redis.Client
	cache *ttlcache.Cache[string, storage.Request]
}

func (c *Cache) InitRequest(ctx context.Context, key string, r storage.Request) error {
	if err := c.rc.RPush(ctx, storage.RecordsList, key).Err(); err != nil {
		return err
	}

	if err := c.rc.Set(ctx, storage.RecordPrefix+key, r, storage.RecordTTL).Err(); err != nil {
		return err
	}

	c.cache.Set(key, r, ttlcache.DefaultTTL)

	return nil
}

func (c *Cache) Set(ctx context.Context, key string, r storage.Request) error {
	if err := c.rc.Set(ctx, storage.RecordPrefix+key, r, storage.RecordTTL).Err(); err != nil {
		return err
	}

	c.cache.Set(key, r, ttlcache.DefaultTTL)

	return nil
}

func (c *Cache) Get(ctx context.Context, key string) (r storage.Request, err error) {
	if item := c.cache.Get(key); item != nil {
		return item.Value(), nil
	}

	if err = c.rc.Get(ctx, storage.RecordPrefix+key).Scan(&r); err != nil {
		return
	}

	c.cache.Set(key, r, ttlcache.DefaultTTL)

	return
}

func (c *Cache) Close() {
	c.cache.Stop()
}

func NewCache(rc *redis.Client) *Cache {
	cache := ttlcache.New(
		ttlcache.WithTTL[string, storage.Request](storage.RecordTTL),
		ttlcache.WithDisableTouchOnHit[string, storage.Request](),
	)

	go cache.Start()

	return &Cache{rc, cache}
}
