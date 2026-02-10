package cache

import (
	"context"

	"github.com/cloud66-oss/geo/utils"
)

type CacheProvider interface {
	Fetch(ctx context.Context, provider string, address string) (*utils.IPInfo, error)
	Add(ctx context.Context, provider string, ipInfo *utils.IPInfo) error
}
