package provider

import (
	"context"

	"github.com/cloud66-oss/geo/utils"
	"github.com/rs/zerolog/log"
)

// CascadeIPProvider is a IPProvider that will try to lookup an IP address in multiple providers
type CascadeIPProvider struct {
	providers    []IPProvider
	stopAtErrors bool
}

func NewCascadeIPProvider(ctx context.Context, stopAtErrors bool, providers []IPProvider) (*CascadeIPProvider, error) {
	return &CascadeIPProvider{
		providers:    providers,
		stopAtErrors: stopAtErrors,
	}, nil
}

func (dpi *CascadeIPProvider) Start(ctx context.Context) error {
	// these should be already started

	return nil
}

func (dpi *CascadeIPProvider) Lookup(ctx context.Context, address string, asFallback bool) (*utils.IPInfo, error) {
	for idx, provider := range dpi.providers {
		ip, err := provider.Lookup(ctx, address, idx != 0)
		if err != nil {
			if dpi.stopAtErrors {
				return nil, err
			} else {
				log.Err(err).Msg("error while looking up IP address, moving on to next provider")
				continue
			}
		}

		if ip != nil {
			return ip, nil
		}
	}

	// not found
	return nil, nil
}

func (dpi *CascadeIPProvider) Shutdown(ctx context.Context) {
	// these should be already shutdown
}

func (dpi *CascadeIPProvider) Refresh(ctx context.Context) error {
	for _, provider := range dpi.providers {
		if err := provider.Refresh(ctx); err != nil {
			return err
		}
	}

	return nil
}
