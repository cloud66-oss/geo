package provider

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/cloud66-oss/geo/utils"
	"github.com/jinzhu/copier"
	"github.com/oschwald/geoip2-golang"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// DbIpProvider is a provider that uses DbIP databases
type DbIpProvider struct {
	cityDb    *geoip2.Reader
	countryDb *geoip2.Reader
	asnDb     *geoip2.Reader
}

func NewDbIpProvider(ctx context.Context) (*DbIpProvider, error) {
	return &DbIpProvider{}, nil
}

func readDbIp(_ context.Context, file string) (*geoip2.Reader, error) {
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

func (mmp *DbIpProvider) Start(ctx context.Context) error {
	log.Info().Msg("starting DbIP Provider")

	if !viper.GetBool("providers.dbip.download.enabled") {
		log.Warn().Msg("DbIP Provider download is disabled, attempting to load existing databases")
		err := mmp.loadDatabases(ctx)
		if err != nil {
			log.Warn().Err(err).Msg("failed to load DbIP databases")
			return nil
		}
		log.Info().Msg("DbIP Provider loaded existing databases successfully")
		return nil
	}

	return mmp.Refresh(ctx)
}

func (mmp *DbIpProvider) downloadDb(_ context.Context, dbName string) error {
	fileURL := viper.GetString(fmt.Sprintf("providers.dbip.download.%s", dbName))
	if fileURL == "" {
		log.Warn().Msg("DbIP Provider fileURL is empty")
		return nil
	}

	filePath := viper.GetString(fmt.Sprintf("providers.dbip.db.%s", dbName))
	if filePath == "" {
		return fmt.Errorf("no local path defined for %s. Use providers.dbip.db.%s to define it", dbName, dbName)
	}

	basePath := filepath.Dir(filePath)
	log.Info().Str("source", fileURL).Str("dest", filePath).Msg("downloading")

	err := os.MkdirAll(basePath, 0700)
	if err != nil {
		return err
	}

	return utils.DownloadFileWithProgress(fileURL, filePath)
}

func (mmp *DbIpProvider) Lookup(ctx context.Context, address string, asFallback bool) (*utils.IPInfo, error) {
	ip := net.ParseIP(address)
	if ip == nil {
		return nil, &utils.IpAddressError{}
	}

	info := &utils.IPInfo{
		Address:            address,
		Source:             "DbIp",
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

	// query the ASN database if available
	if mmp.asnDb != nil {
		asn, err := mmp.asnDb.ASN(ip)
		if err != nil {
			return nil, err
		}

		err = copier.Copy(&info.ASN, &asn)
		if err != nil {
			return nil, err
		}
		info.HasASN = true
	} else {
		info.HasASN = false
	}

	if mmp.cityDb != nil {
		// city db includes country data as well
		city, err := mmp.cityDb.City(ip)
		if err != nil {
			return nil, err
		}

		err = copier.Copy(&info, &city)
		if err != nil {
			return nil, err
		}

		info.HasCity = true
	} else if mmp.countryDb != nil {
		// fall back to country-only db when no city db is available
		country, err := mmp.countryDb.Country(ip)
		if err != nil {
			return nil, err
		}

		err = copier.Copy(&info, &country)
		if err != nil {
			return nil, err
		}

		info.HasCity = false
	}

	info.HasAnonymousIP = false

	return info, nil
}

func (mmp *DbIpProvider) Shutdown(ctx context.Context) {
	if mmp.cityDb != nil {
		mmp.cityDb.Close()
	}
	if mmp.countryDb != nil {
		mmp.countryDb.Close()
	}
	if mmp.asnDb != nil {
		mmp.asnDb.Close()
	}
}

func (mmp *DbIpProvider) Refresh(ctx context.Context) error {
	log.Info().Msg("refreshing DbIP Provider")
	if err := mmp.downloadDb(ctx, "city"); err != nil {
		return err
	}

	if err := mmp.downloadDb(ctx, "asn"); err != nil {
		return err
	}

	// download the country database
	if err := mmp.downloadDb(ctx, "country"); err != nil {
		return err
	}

	return mmp.loadDatabases(ctx)
}

func (mmp *DbIpProvider) loadDatabases(ctx context.Context) error {
	// load the city database
	db, err := readDbIp(ctx, viper.GetString("providers.dbip.db.city"))
	if err != nil {
		return err
	}
	mmp.cityDb = db

	// load the country database
	db, err = readDbIp(ctx, viper.GetString("providers.dbip.db.country"))
	if err != nil {
		return err
	}
	mmp.countryDb = db

	// load the ASN database
	db, err = readDbIp(ctx, viper.GetString("providers.dbip.db.asn"))
	if err != nil {
		return err
	}
	mmp.asnDb = db

	return nil
}
