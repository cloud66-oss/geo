package provider

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/jinzhu/copier"
	"github.com/oschwald/geoip2-golang"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

	"github.com/cloud66-oss/geo/utils"
)

// GlobioProvider is a provider that uses Globio databases (country and ASN)
type GlobioProvider struct {
	countryDb *geoip2.Reader
	asnDb     *geoip2.Reader
}

func NewGlobioProvider(ctx context.Context) (*GlobioProvider, error) {
	return &GlobioProvider{}, nil
}

func readGlobioDb(_ context.Context, file string) (*geoip2.Reader, error) {
	if file == "" {
		return nil, nil
	}

	if !utils.FileExists(file) {
		return nil, fmt.Errorf("file not found %s", file)
	}

	db, err := geoip2.Open(file)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func (gp *GlobioProvider) Start(ctx context.Context) error {
	log.Info().Msg("starting Globio Provider")

	if !viper.GetBool("providers.globio.download.enabled") {
		log.Warn().Msg("Globio Provider download is disabled, attempting to load existing databases")
		err := gp.loadDatabases(ctx)
		if err != nil {
			log.Warn().Err(err).Msg("failed to load Globio databases")
			return nil
		}
		log.Info().Msg("Globio Provider loaded existing databases successfully")
		return nil
	}

	return gp.Refresh(ctx)
}

func (gp *GlobioProvider) downloadDb(_ context.Context, dbName string) error {
	fileURL := viper.GetString(fmt.Sprintf("providers.globio.download.%s", dbName))
	if fileURL == "" {
		log.Warn().Msg("Globio Provider fileURL is empty")
		return nil
	}

	filePath := viper.GetString(fmt.Sprintf("providers.globio.db.%s", dbName))
	if filePath == "" {
		return fmt.Errorf("no local path defined for %s. Use providers.globio.db.%s to define it", dbName, dbName)
	}

	basePath := filepath.Dir(filePath)
	log.Info().Str("source", fileURL).Str("dest", filePath).Msg("downloading")

	err := os.MkdirAll(basePath, 0700)
	if err != nil {
		return err
	}

	return utils.DownloadFileWithProgress(fileURL, filePath)
}

func (gp *GlobioProvider) Lookup(ctx context.Context, address string, asFallback bool) (*utils.IPInfo, error) {
	ip := net.ParseIP(address)
	if ip == nil {
		return nil, &utils.IpAddressError{}
	}

	info := &utils.IPInfo{
		Address:            address,
		Source:             "globio",
		IsFallback:         asFallback,
		ASN:                &utils.ASN{},
		Location:           &utils.Location{},
		AnonymousIP:        &utils.AnonymousIP{},
		City:               &utils.City{},
		Continent:          &utils.Continent{},
		Country:            &utils.Country{},
		Postal:             &utils.Postal{},
		RegisteredCountry:  &utils.Country{},
		RepresentedCountry: &utils.Country{},
		Subdivisions:       []*utils.Subdivision{},
		Traits:             &utils.Traits{},
	}

	if gp.countryDb != nil {
		country, err := gp.countryDb.Country(ip)
		if err != nil {
			return nil, err
		}

		err = copier.Copy(&info, &country)
		if err != nil {
			return nil, err
		}
	}

	// query the ASN database if available
	if gp.asnDb != nil {
		asn, err := gp.asnDb.ASN(ip)
		if err != nil {
			return nil, err
		}

		err = copier.Copy(&info.ASN, &asn)
		if err != nil {
			return nil, err
		}
		info.HasASN = true
	}

	// globio does not have city or anonymous IP data
	info.HasCity = false
	info.HasAnonymousIP = false

	return info, nil
}

func (gp *GlobioProvider) Shutdown(ctx context.Context) {
	if gp.countryDb != nil {
		gp.countryDb.Close()
	}
	if gp.asnDb != nil {
		gp.asnDb.Close()
	}
}

func (gp *GlobioProvider) Refresh(ctx context.Context) error {
	log.Info().Msg("refreshing Globio Provider")
	if err := gp.downloadDb(ctx, "country"); err != nil {
		return err
	}

	// download the ASN database
	if err := gp.downloadDb(ctx, "asn"); err != nil {
		return err
	}

	return gp.loadDatabases(ctx)
}

func (gp *GlobioProvider) loadDatabases(ctx context.Context) error {
	// load the country database
	db, err := readGlobioDb(ctx, viper.GetString("providers.globio.db.country"))
	if err != nil {
		return err
	}
	gp.countryDb = db

	// load the ASN database
	db, err = readGlobioDb(ctx, viper.GetString("providers.globio.db.asn"))
	if err != nil {
		return err
	}
	gp.asnDb = db

	return nil
}
