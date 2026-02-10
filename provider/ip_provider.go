package provider

import (
	"context"

	"github.com/cloud66-oss/geo/utils"
)

type IPProvider interface {
	Start(ctx context.Context) error
	Lookup(ctx context.Context, address string, asFallback bool) (*utils.IPInfo, error)
	Shutdown(ctx context.Context)
	Refresh(ctx context.Context) error
}
