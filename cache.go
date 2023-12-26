//go:build go1.18

package brick

import (
	"context"
	"time"

	"github.com/bool64/cache"
	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
)

// TransferCache performs cache transfer.
func (l *BaseLocator) TransferCache(ctx context.Context) error {
	if l.BaseConfig.CacheTransferURL == "" || l.cacheTransfer.CachesCount() == 0 {
		return nil
	}

	return l.cacheTransfer.Import(ctx, l.BaseConfig.CacheTransferURL)
}

// MakeCacheOf creates an instance of failover cache and adds it to cache transfer.
func MakeCacheOf[V any](l interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger
}, name string, ttl time.Duration, options ...func(cfg *cache.FailoverConfigOf[V]),
) *cache.FailoverOf[V] {
	cfg := cache.FailoverConfigOf[V]{}
	cfg.Name = name
	cfg.Stats = l.StatsTracker()
	cfg.Logger = l.CtxdLogger()

	for _, option := range options {
		option(&cfg)
	}

	if cfg.Backend == nil {
		cfg.Backend = cache.NewShardedMapOf[V](func(cfg *cache.Config) {
			cfg.Name = name
			cfg.Logger = l.CtxdLogger()
			cfg.Stats = l.StatsTracker()
			cfg.TimeToLive = ttl
		})
	}

	fc := cache.NewFailoverOf[V](func(c *cache.FailoverConfigOf[V]) {
		*c = cfg
	})

	if l, ok := l.(interface {
		CacheTransfer() *cache.HTTPTransfer
	}); ok {
		if w, ok := cfg.Backend.(cache.WalkDumpRestorer); ok {
			l.CacheTransfer().AddCache(name, w)
		}
	}

	if l, ok := l.(interface {
		CacheInvalidationIndex() *cache.InvalidationIndex
	}); ok {
		if d, ok := cfg.Backend.(cache.Deleter); ok {
			l.CacheInvalidationIndex().AddCache(name, d)
		}
	}

	return fc
}
