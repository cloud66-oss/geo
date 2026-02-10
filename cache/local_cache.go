package cache

import (
	"context"

	"github.com/cloud66-oss/geo/utils"
	lru "github.com/hashicorp/golang-lru"
	"github.com/spf13/viper"
)

type LocalCache struct {
	cache *lru.ARCCache
}

func NewLocalCache(ctx context.Context) (*LocalCache, error) {
	cache, err := lru.NewARC(viper.GetInt("cache.size"))
	if err != nil {
		return nil, err
	}

	return &LocalCache{
		cache: cache,
	}, nil
}

func (lc *LocalCache) Fetch(ctx context.Context, provider string, address string) (*utils.IPInfo, error) {
	value, ok := lc.cache.Get(provider + "--" + address)
	if ok {
		return value.(*utils.IPInfo), nil
	}

	return nil, nil
}

func (lc *LocalCache) Add(ctx context.Context, provider string, ipInfo *utils.IPInfo) error {
	lc.cache.Add(provider+"--"+ipInfo.Address, ipInfo)

	return nil
}
