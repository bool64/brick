//go:build go1.18

package brick

import (
	"context"
	"time"

	"github.com/bool64/cache"
)

// TransferCache performs cache transfer.
func (l *BaseLocator) TransferCache(ctx context.Context) error {
	if l.BaseConfig.CacheTransferURL == "" || l.CacheTransfer.CachesCount() == 0 {
		return nil
	}

	return l.CacheTransfer.Import(ctx, l.BaseConfig.CacheTransferURL)
}

// MakeCacheOf creates an instance of failover cache and adds it to cache transfer.
func MakeCacheOf[V any](l *BaseLocator, name string, ttl time.Duration) *cache.FailoverOf[V] {
	backend := cache.NewShardedMapOf[V](func(cfg *cache.Config) {
		cfg.Name = name
		cfg.Logger = l.CtxdLogger()
		cfg.Stats = l.StatsTracker()
		cfg.TimeToLive = ttl
	})

	greetingsCache := cache.NewFailoverOf[V](func(cfg *cache.FailoverConfigOf[V]) {
		cfg.Name = name
		cfg.Stats = l.StatsTracker()
		cfg.Logger = l.CtxdLogger()
		cfg.Backend = backend
	})

	l.CacheTransfer.AddCache(name, backend.WalkDumpRestorer())

	return greetingsCache
}
