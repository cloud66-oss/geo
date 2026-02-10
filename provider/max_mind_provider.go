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

// MaxMindProvider is a provider that uses MaxMind databases
type MaxMindProvider struct {
	cityDb      *geoip2.Reader
	asnDb       *geoip2.Reader
	anonymousDb *geoip2.Reader
}

func NewMaxMindProvider(ctx context.Context) (*MaxMindProvider, error) {
	return &MaxMindProvider{}, nil
}

func readMaxMindDb(_ context.Context, file string) (*geoip2.Reader, error) {
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

func (mmp *MaxMindProvider) Start(ctx context.Context) error {
	log.Info().Msg("starting MaxMind Provider")

	if !viper.GetBool("providers.maxmind.download.enabled") {
		log.Warn().Msg("MaxMind Provider download is disabled, attempting to load existing databases")
		err := mmp.loadDatabases(ctx)
		if err != nil {
			log.Warn().Err(err).Msg("failed to load MaxMind databases")
			return nil
		}
		log.Info().Msg("MaxMind Provider loaded existing databases successfully")
		return nil
	}

	return mmp.Refresh(ctx)
}

func (mmp *MaxMindProvider) downloadDb(_ context.Context, dbName string) error {
	filePath := viper.GetString(fmt.Sprintf("providers.maxmind.db.%s", dbName))
	if filePath == "" {
		log.Debug().Str("db", dbName).Msg("no local path defined, skipping download")
		return nil
	}

	basePath := filepath.Dir(filePath)
	err := os.MkdirAll(basePath, 0700)
	if err != nil {
		return err
	}

	// Direct download from MaxMind API when license_key is configured
	licenseKey := viper.GetString("providers.maxmind.license_key")
	if licenseKey != "" {
		accountID := viper.GetString("providers.maxmind.account_id")
		editionID := viper.GetString(fmt.Sprintf("providers.maxmind.editions.%s", dbName))
		if editionID == "" {
			log.Debug().Str("db", dbName).Msg("no edition ID configured, skipping")
			return nil
		}

		log.Info().Str("edition", editionID).Str("dest", filePath).Msg("downloading from MaxMind")
		return utils.DownloadMaxMindDb(accountID, licenseKey, editionID, filePath)
	}

	// Fallback: download from configured URL (e.g. GCS mirror)
	fileURL := viper.GetString(fmt.Sprintf("providers.maxmind.download.%s", dbName))
	if fileURL == "" {
		log.Warn().Str("db", dbName).Msg("no download URL configured")
		return nil
	}

	log.Info().Str("source", fileURL).Str("dest", filePath).Msg("downloading")
	return utils.DownloadFileWithProgress(fileURL, filePath)
}

func (mmp *MaxMindProvider) Lookup(ctx context.Context, address string, asFallback bool) (*utils.IPInfo, error) {
	ip := net.ParseIP(address)
	if ip == nil {
		return nil, &utils.IpAddressError{}
	}

	info := &utils.IPInfo{
		Address:            address,
		Source:             "maxmind",
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
	}

	if mmp.cityDb != nil {
		city, err := mmp.cityDb.City(ip)
		if err != nil {
			return nil, err
		}

		err = copier.Copy(&info, &city)
		if err != nil {
			return nil, err
		}

		info.HasCity = true
	}

	if mmp.anonymousDb != nil {
		anon, err := mmp.anonymousDb.AnonymousIP(ip)
		if err != nil {
			return nil, err
		}

		err = copier.Copy(&info.AnonymousIP, &anon)
		if err != nil {
			return nil, err
		}

		info.HasAnonymousIP = true
	}

	return info, nil
}

func (mmp *MaxMindProvider) Shutdown(ctx context.Context) {
	if mmp.cityDb != nil {
		mmp.cityDb.Close()
	}
	if mmp.asnDb != nil {
		mmp.asnDb.Close()
	}
}

func (mmp *MaxMindProvider) Refresh(ctx context.Context) error {
	log.Info().Msg("refreshing MaxMind Provider")
	if err := mmp.downloadDb(ctx, "city"); err != nil {
		return err
	}

	if err := mmp.downloadDb(ctx, "asn"); err != nil {
		return err
	}

	if err := mmp.downloadDb(ctx, "anonymous"); err != nil {
		return err
	}

	return mmp.loadDatabases(ctx)
}

func (mmp *MaxMindProvider) loadDatabases(ctx context.Context) error {
	db, err := readMaxMindDb(ctx, viper.GetString("providers.maxmind.db.city"))
	if err != nil {
		return err
	}
	mmp.cityDb = db

	db, err = readMaxMindDb(ctx, viper.GetString("providers.maxmind.db.asn"))
	if err != nil {
		return err
	}
	mmp.asnDb = db

	db, err = readMaxMindDb(ctx, viper.GetString("providers.maxmind.db.anonymous"))
	if err != nil {
		return err
	}
	mmp.anonymousDb = db

	return nil
}
