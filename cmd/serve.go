package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/cloud66-oss/geo/cache"
	"github.com/cloud66-oss/geo/provider"
	"github.com/cloud66-oss/geo/utils"
	"github.com/getsentry/sentry-go"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serveCmd = &cobra.Command{
	Use: "serve",
	Run: execServe,
}

func init() {
	// api server
	serveCmd.PersistentFlags().String("binding", "0.0.0.0", "API binding")
	serveCmd.PersistentFlags().Int("port", 9912, "API port")

	serveCmd.PersistentFlags().String("default", "maxmind", "Default IP provider")

	serveCmd.PersistentFlags().String("providers.maxmind.db.city", "", "MaxMind city database")
	serveCmd.PersistentFlags().String("providers.maxmind.db.asn", "", "MaxMind ASN database")
	serveCmd.PersistentFlags().Bool("providers.maxmind.download.enabled", false, "MaxMind download enabled")
	serveCmd.PersistentFlags().String("providers.maxmind.download.city", "", "MaxMind download city database URL")
	serveCmd.PersistentFlags().String("providers.maxmind.download.asn", "", "MaxMind download ASN database URL")
	serveCmd.PersistentFlags().String("providers.maxmind.account_id", "", "MaxMind account ID")
	serveCmd.PersistentFlags().String("providers.maxmind.license_key", "", "MaxMind license key")
	serveCmd.PersistentFlags().String("providers.maxmind.editions.city", "GeoLite2-City", "MaxMind city edition ID")
	serveCmd.PersistentFlags().String("providers.maxmind.editions.asn", "GeoLite2-ASN", "MaxMind ASN edition ID")
	serveCmd.PersistentFlags().String("providers.maxmind.editions.anonymous", "", "MaxMind anonymous IP edition ID")
	serveCmd.PersistentFlags().Bool("providers.maxmind.enabled", false, "MaxMind enabled")

	serveCmd.PersistentFlags().String("providers.dbip.db.city", "", "DbIp city database")
	serveCmd.PersistentFlags().String("providers.dbip.db.asn", "", "DbIp ASN database")
	serveCmd.PersistentFlags().String("providers.dbip.db.country", "", "DbIp country database")
	serveCmd.PersistentFlags().Bool("providers.dbip.download.enabled", false, "DbIp download enabled")
	serveCmd.PersistentFlags().String("providers.dbip.download.city", "", "DbIp download city database URL")
	serveCmd.PersistentFlags().String("providers.dbip.download.asn", "", "DbIp download ASN database URL")
	serveCmd.PersistentFlags().String("providers.dbip.download.country", "", "DbIp download country database URL")
	serveCmd.PersistentFlags().Bool("providers.dbip.enabled", false, "DbIp enabled")

	serveCmd.PersistentFlags().String("providers.ipstack.apikey", "", "IPStack API key")
	serveCmd.PersistentFlags().Bool("providers.ipstack.enabled", false, "IPStack enabled")

	serveCmd.PersistentFlags().String("providers.globio.db.country", "", "Globio country database")
	serveCmd.PersistentFlags().String("providers.globio.db.asn", "", "Globio ASN database")
	serveCmd.PersistentFlags().Bool("providers.globio.download.enabled", false, "Globio download enabled")
	serveCmd.PersistentFlags().String("providers.globio.download.country", "", "Globio download country database URL")
	serveCmd.PersistentFlags().String("providers.globio.download.asn", "", "Globio download ASN database URL")
	serveCmd.PersistentFlags().Bool("providers.globio.enabled", false, "Globio enabled")

	serveCmd.PersistentFlags().Bool("providers.cascade.enabled", false, "Cascade enabled")
	serveCmd.PersistentFlags().StringArray("providers.cascade.providers", []string{"maxmind", "ipstack"}, "Cascade providers")
	serveCmd.PersistentFlags().Bool("providers.cascade.stopOnError", false, "Cascade stop on error")

	viper.BindPFlag("default", serveCmd.PersistentFlags().Lookup("default"))
	viper.BindPFlag("api.binding", serveCmd.PersistentFlags().Lookup("binding"))
	viper.BindPFlag("api.port", serveCmd.PersistentFlags().Lookup("port"))

	viper.BindPFlag("providers.maxmind.db.city", serveCmd.PersistentFlags().Lookup("providers.maxmind.db.city"))
	viper.BindPFlag("providers.maxmind.db.asn", serveCmd.PersistentFlags().Lookup("providers.maxmind.db.asn"))
	viper.BindPFlag("providers.maxmind.download.enabled", serveCmd.PersistentFlags().Lookup("providers.maxmind.download.enabled"))
	viper.BindPFlag("providers.maxmind.download.city", serveCmd.PersistentFlags().Lookup("providers.maxmind.download.city"))
	viper.BindPFlag("providers.maxmind.download.asn", serveCmd.PersistentFlags().Lookup("providers.maxmind.download.asn"))
	viper.BindPFlag("providers.maxmind.account_id", serveCmd.PersistentFlags().Lookup("providers.maxmind.account_id"))
	viper.BindPFlag("providers.maxmind.license_key", serveCmd.PersistentFlags().Lookup("providers.maxmind.license_key"))
	viper.BindPFlag("providers.maxmind.editions.city", serveCmd.PersistentFlags().Lookup("providers.maxmind.editions.city"))
	viper.BindPFlag("providers.maxmind.editions.asn", serveCmd.PersistentFlags().Lookup("providers.maxmind.editions.asn"))
	viper.BindPFlag("providers.maxmind.editions.anonymous", serveCmd.PersistentFlags().Lookup("providers.maxmind.editions.anonymous"))
	viper.BindPFlag("providers.maxmind.enabled", serveCmd.PersistentFlags().Lookup("providers.maxmind.enabled"))

	viper.BindPFlag("providers.dbip.db.city", serveCmd.PersistentFlags().Lookup("providers.dbip.db.city"))
	viper.BindPFlag("providers.dbip.db.asn", serveCmd.PersistentFlags().Lookup("providers.dbip.db.asn"))
	viper.BindPFlag("providers.dbip.db.country", serveCmd.PersistentFlags().Lookup("providers.dbip.db.country"))
	viper.BindPFlag("providers.dbip.download.enabled", serveCmd.PersistentFlags().Lookup("providers.dbip.download.enabled"))
	viper.BindPFlag("providers.dbip.download.city", serveCmd.PersistentFlags().Lookup("providers.dbip.download.city"))
	viper.BindPFlag("providers.dbip.download.asn", serveCmd.PersistentFlags().Lookup("providers.dbip.download.asn"))
	viper.BindPFlag("providers.dbip.download.country", serveCmd.PersistentFlags().Lookup("providers.dbip.download.country"))
	viper.BindPFlag("providers.dbip.enabled", serveCmd.PersistentFlags().Lookup("providers.dbip.enabled"))

	viper.BindPFlag("providers.ipstack.enabled", serveCmd.PersistentFlags().Lookup("providers.ipstack.enabled"))

	viper.BindPFlag("providers.globio.db.country", serveCmd.PersistentFlags().Lookup("providers.globio.db.country"))
	viper.BindPFlag("providers.globio.db.asn", serveCmd.PersistentFlags().Lookup("providers.globio.db.asn"))
	viper.BindPFlag("providers.globio.download.enabled", serveCmd.PersistentFlags().Lookup("providers.globio.download.enabled"))
	viper.BindPFlag("providers.globio.download.country", serveCmd.PersistentFlags().Lookup("providers.globio.download.country"))
	viper.BindPFlag("providers.globio.download.asn", serveCmd.PersistentFlags().Lookup("providers.globio.download.asn"))
	viper.BindPFlag("providers.globio.enabled", serveCmd.PersistentFlags().Lookup("providers.globio.enabled"))

	viper.BindPFlag("providers.cascade.enabled", serveCmd.PersistentFlags().Lookup("providers.cascade.enabled"))
	viper.BindPFlag("providers.cascade.stopOnError", serveCmd.PersistentFlags().Lookup("providers.cascade.stopOnError"))
	viper.BindPFlag("providers.cascade.providers", serveCmd.PersistentFlags().Lookup("providers.cascade.providers"))

	// providers
	viper.SetDefault("providers.maxmind.db.city", "")
	viper.SetDefault("providers.maxmind.db.asn", "")
	viper.SetDefault("providers.maxmind.download.enabled", false)
	viper.SetDefault("providers.maxmind.download.city", "")
	viper.SetDefault("providers.maxmind.download.asn", "")
	viper.SetDefault("providers.maxmind.account_id", "")
	viper.SetDefault("providers.maxmind.license_key", "")
	viper.SetDefault("providers.maxmind.editions.city", "GeoLite2-City")
	viper.SetDefault("providers.maxmind.editions.asn", "GeoLite2-ASN")
	viper.SetDefault("providers.maxmind.editions.anonymous", "")
	viper.SetDefault("providers.maxmind.enabled", false)

	viper.SetDefault("providers.dbip.db.city", "")
	viper.SetDefault("providers.dbip.db.asn", "")
	viper.SetDefault("providers.dbip.db.country", "")
	viper.SetDefault("providers.dbip.download.enabled", false)
	viper.SetDefault("providers.dbip.download.city", "")
	viper.SetDefault("providers.dbip.download.asn", "")
	viper.SetDefault("providers.dbip.download.country", "")
	viper.SetDefault("providers.dbip.enabled", false)

	viper.SetDefault("providers.ipstack.apikey", "")
	viper.SetDefault("providers.ipstack.enabled", false)

	viper.SetDefault("providers.globio.db.country", "")
	viper.SetDefault("providers.globio.db.asn", "")
	viper.SetDefault("providers.globio.download.enabled", false)
	viper.SetDefault("providers.globio.download.country", "")
	viper.SetDefault("providers.globio.download.asn", "")
	viper.SetDefault("providers.globio.enabled", false)

	viper.SetDefault("providers.cascade.enabled", false)
	viper.SetDefault("providers.cascade.stopOnError", false)
	viper.SetDefault("providers.cascade.providers", []string{"maxmind", "ipstack"})

	// cache
	viper.SetDefault("cache.enabled", true)
	viper.SetDefault("cache.size", 128)

	// refresh
	viper.SetDefault("refresh", "24h")

	rootCmd.AddCommand(serveCmd)
}

func getIP(c echo.Context) error {
	cached := viper.GetBool("cache.enabled")

	var cp cache.CacheProvider
	if cached {
		cp = utils.Container.Fetch(c.Request().Context(), utils.Cache).(cache.CacheProvider)
	}

	requestedProvider := c.QueryParam("provider")
	if requestedProvider == "" {
		requestedProvider = viper.GetString("default")
	}

	address := c.Param("address")
	log.Debug().Str("address", address).Str("provider", requestedProvider).Msg("fetching")

	if cached {
		ip, err := cp.Fetch(c.Request().Context(), requestedProvider, address)
		if err != nil {
			log.Error().Err(err).Msg("failed to fetch from cache")
		}

		if ip != nil {
			log.Trace().Str("address", address).Msg("returning cached value")
			return c.JSON(http.StatusOK, ip)
		}
	}

	if cached {
		log.Trace().Str("address", address).Str("provider", requestedProvider).Msg("not found in cache")
	}

	ipProvider, err := getRequestedProvider(c.Request().Context(), requestedProvider)
	if err != nil {
		if _, ok := err.(*utils.UnknownProviderError); ok {
			return c.JSON(http.StatusBadRequest, utils.ErrorResponse{
				Error: err.Error(),
			})
		} else {
			log.Err(err).Msg("failed to get provider")
			return c.JSON(http.StatusInternalServerError, err)
		}
	}

	ip, err := ipProvider.Lookup(c.Request().Context(), address, false)
	if err != nil {
		if ipErr, ok := err.(*utils.IpAddressError); ok {
			return c.JSON(http.StatusBadRequest, utils.ErrorResponse{
				Error: ipErr.Error(),
			})
		} else {
			log.Error().Str("address", address).Str("provider", requestedProvider).Err(err).Msg("failed to lookup ip address")
			sentry.CaptureException(err)
			return err
		}
	}

	if cached {
		log.Trace().Str("address", address).Msg("adding to cache")
		if err := cp.Add(c.Request().Context(), requestedProvider, ip); err != nil {
			log.Error().Err(err).Msg("failed to update cache")
		}
	}

	return c.JSON(http.StatusOK, ip)
}

func getRequestedProvider(ctx context.Context, name string) (provider.IPProvider, error) {
	switch name {
	case "maxmind":
		return utils.Container.Fetch(ctx, utils.MaxMindProvider).(provider.IPProvider), nil
	case "dbip":
		return utils.Container.Fetch(ctx, utils.DbIpProvider).(provider.IPProvider), nil
	case "ipstack":
		return utils.Container.Fetch(ctx, utils.IpStackProvider).(provider.IPProvider), nil
	case "globio":
		return utils.Container.Fetch(ctx, utils.GlobioProvider).(provider.IPProvider), nil
	default:
		return nil, &utils.UnknownProviderError{}
	}
}

func isProviderEnabled(name string) bool {
	switch name {
	case "maxmind":
		return viper.GetBool("providers.maxmind.enabled")
	case "dbip":
		return viper.GetBool("providers.dbip.enabled")
	case "ipstack":
		return viper.GetBool("providers.ipstack.enabled")
	case "globio":
		return viper.GetBool("providers.globio.enabled")
	case "cascade":
		return viper.GetBool("providers.cascade.enabled")
	default:
		return false
	}
}

func getEnabledProviders(ctx context.Context) []provider.IPProvider {
	var providers []provider.IPProvider

	if viper.GetBool("providers.maxmind.enabled") {
		providers = append(providers, utils.Container.Fetch(ctx, utils.MaxMindProvider).(provider.IPProvider))
	}

	if viper.GetBool("providers.dbip.enabled") {
		providers = append(providers, utils.Container.Fetch(ctx, utils.DbIpProvider).(provider.IPProvider))
	}

	if viper.GetBool("providers.ipstack.enabled") {
		providers = append(providers, utils.Container.Fetch(ctx, utils.IpStackProvider).(provider.IPProvider))
	}

	if viper.GetBool("providers.globio.enabled") {
		providers = append(providers, utils.Container.Fetch(ctx, utils.GlobioProvider).(provider.IPProvider))
	}

	return providers
}

func configureCache(ctx context.Context) error {
	cache, err := cache.NewLocalCache(ctx)
	if err != nil {
		return err
	}

	utils.Container.Assign(ctx, utils.Cache, cache)

	return nil
}

func execServe(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	// which provider
	defaultProviderName := viper.GetString("default")
	log.Info().Str("default", defaultProviderName).Msg("using provider as default")

	// validate that the default provider is enabled
	if !isProviderEnabled(defaultProviderName) {
		log.Fatal().Str("provider", defaultProviderName).Msg("default provider is not enabled")
	}

	if viper.GetBool("providers.maxmind.enabled") {
		ipProvider, err := provider.NewMaxMindProvider(ctx)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to open maxmind provider")
		}

		utils.Container.Assign(ctx, utils.MaxMindProvider, ipProvider)
		err = ipProvider.Start(ctx)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to start maxmind provider")
		}
	}

	if viper.GetBool("providers.dbip.enabled") {
		ipProvider, err := provider.NewDbIpProvider(ctx)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to open dbip provider")
		}

		utils.Container.Assign(ctx, utils.DbIpProvider, ipProvider)
		err = ipProvider.Start(ctx)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to start dbip provider")
		}
	}
	if viper.GetBool("providers.ipstack.enabled") {
		ipProvider, err := provider.NewIpStackProvider(ctx)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to open ipstack provider")
		}

		utils.Container.Assign(ctx, utils.IpStackProvider, ipProvider)
		err = ipProvider.Start(ctx)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to start ipstack provider")
		}
	}

	if viper.GetBool("providers.globio.enabled") {
		ipProvider, err := provider.NewGlobioProvider(ctx)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to open globio provider")
		}

		utils.Container.Assign(ctx, utils.GlobioProvider, ipProvider)
		err = ipProvider.Start(ctx)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to start globio provider")
		}
	}

	// this should always be the last and all used providers should be enabled
	if viper.GetBool("providers.cascade.enabled") {
		providers := make([]provider.IPProvider, 0)
		for _, providerName := range viper.GetStringSlice("providers.cascade.providers") {
			log.Info().Str("provider", providerName).Msg("adding provider to cascade")
			switch providerName {
			case "maxmind":
				providers = append(providers, utils.Container.Fetch(ctx, utils.MaxMindProvider).(provider.IPProvider))
			case "dbip":
				providers = append(providers, utils.Container.Fetch(ctx, utils.DbIpProvider).(provider.IPProvider))
			case "ipstack":
				providers = append(providers, utils.Container.Fetch(ctx, utils.IpStackProvider).(provider.IPProvider))
			case "globio":
				providers = append(providers, utils.Container.Fetch(ctx, utils.GlobioProvider).(provider.IPProvider))
			default:
				log.Fatal().Str("provider", providerName).Msg("unknown provider")
			}

			ipProvider, err := provider.NewCascadeIPProvider(ctx, viper.GetBool("providers.cascade.stopOnError"), providers)
			if err != nil {
				log.Fatal().Err(err).Msg("failed to open cascade provider")
			}

			err = ipProvider.Start(ctx)
			if err != nil {
				log.Fatal().Err(err).Msg("failed to start ip provider")
			}
		}
	}

	if viper.GetBool("cache.enabled") {
		err := configureCache(ctx)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to start cache")
		}
	}

	err := startServer(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to start the api server")
	}
}

func ping(c echo.Context) error {
	return c.String(http.StatusOK, "pong")
}

func startServer(ctx context.Context) error {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(middleware.RequestID())
	e.Use(utils.ZeroLogger(&log.Logger))
	e.GET("/_ping", ping)
	e.GET("/v1/ip/:address", getIP)

	go func() {
		if err := e.Start(fmt.Sprintf("%s:%d", viper.GetString("api.binding"), viper.GetInt("api.port"))); err != nil {
			if err != http.ErrServerClosed {
				log.Error().Err(err).Msg("failed to start the server")
			}
		}
	}()

	stopRefresh := make(chan bool)
	// refresh in intervals
	ticker := time.NewTicker(time.Duration(viper.GetDuration("refresh")))
	go func() {
		for {
			select {
			case <-ticker.C:
				log.Info().Msg("refreshing providers")
				for _, provider := range getEnabledProviders(ctx) {
					err := provider.Refresh(ctx)
					if err != nil {
						log.Error().Err(err).Msg("failed to refresh provider")
					}
				}
			case <-stopRefresh:
				log.Info().Msg("stopping refresh")
				return
			}
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	stopRefresh <- true

	// shutdown the all enabled providers
	for _, provider := range getEnabledProviders(ctx) {
		provider.Shutdown(ctx)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}

	return nil
}
