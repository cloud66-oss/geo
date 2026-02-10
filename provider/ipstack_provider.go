package provider

import (
	"context"
	"strconv"

	"github.com/cloud66-oss/geo/utils"
	"github.com/qioalice/ipstack"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type IpStackProvider struct {
	cli *ipstack.Client
}

func NewIpStackProvider(ctx context.Context) (*IpStackProvider, error) {
	return &IpStackProvider{}, nil
}

func (provider *IpStackProvider) Start(ctx context.Context) error {
	log.Info().Msg("starting IpStack Provider")

	cli, err := ipstack.New(
		ipstack.ParamToken(viper.GetString("providers.ipstack.apikey")),
		ipstack.ParamUseHTTPS(true),
	)

	if err != nil {
		log.Info().Msg("failed to create IpStack client. Have you remembered to set the API key? You can use the GEO_PROVIDERS_IPSTACK_APIKEY environment variable or providers.ipstack.apikey in the config file or as a param")
		return err
	}

	provider.cli = cli

	return nil
}

func (provider *IpStackProvider) Lookup(ctx context.Context, address string, asFallback bool) (*utils.IPInfo, error) {
	ipInfo, err := provider.cli.IP(address)

	if err != nil {
		log.Error().Err(err).Msg("failed to lookup IP address for IPStack")
		return nil, err
	}

	callingCode, _ := strconv.Atoi(ipInfo.Location.CallingCode)
	country := &utils.Country{
		IsInEuropeanUnion: ipInfo.Location.IsEU,
		IsoCode:           ipInfo.CountryCode,
		Names: map[string]string{
			"en": ipInfo.CountryName,
		},
	}

	info := &utils.IPInfo{
		Address:    ipInfo.IP,
		Source:     "ipstack",
		HasCity:    true,
		IsFallback: asFallback,
		City: &utils.City{
			GeoNameID: uint(ipInfo.Location.GeonameID),
			Names: map[string]string{
				"en": ipInfo.City,
			},
		},
		Continent: &utils.Continent{
			Code: ipInfo.ContinentCode,
			Names: map[string]string{
				"en": ipInfo.ContinentName,
			},
		},
		Country:            country,
		RegisteredCountry:  country,
		RepresentedCountry: country,
		Location: &utils.Location{
			Latitude:  float64(ipInfo.Latitide),
			Longitude: float64(ipInfo.Longitude),
			TimeZone:  ipInfo.Timezone.ID,
			MetroCode: uint(callingCode),
		},
		Postal: &utils.Postal{
			Code: ipInfo.Zip,
		},
		Subdivisions: []*utils.Subdivision{
			{
				IsoCode: ipInfo.RegionCode,
				Names: map[string]string{
					"en": ipInfo.RegionName,
				},
			},
		},
		HasASN: true,
		ASN: &utils.ASN{
			AutonomousSystemNumber:       uint(ipInfo.Connection.ASN),
			AutonomousSystemOrganization: ipInfo.Connection.ISP,
		},
	}

	return info, nil
}

func (provider *IpStackProvider) Shutdown(ctx context.Context) {
	log.Info().Msg("shutting down IpStack Provider")
}

func (provider *IpStackProvider) Refresh(ctx context.Context) error {
	return nil
}
