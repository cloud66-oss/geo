package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cloud66-oss/geo/utils"
	"github.com/fsnotify/fsnotify"
	"github.com/getsentry/sentry-go"
	"github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "geo",
	Short: "geo is a GeoIP Microservice",
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.Version = utils.Version
	rootCmd.SetVersionTemplate("{{.Version}}\n")

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/geo.yml)")
	rootCmd.PersistentFlags().String("level", "info", "log level")
	rootCmd.PersistentFlags().String("log-format", "json", "log format: json or text")

	viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("level"))
	viper.BindPFlag("log.format", rootCmd.PersistentFlags().Lookup("log-format"))

	rootCmd.AddCommand(serveCmd)
}

func configureLogging(_ context.Context) {
	level, err := zerolog.ParseLevel(viper.GetString("log.level"))
	if err != nil {
		fmt.Println("invalid log level")
		os.Exit(1)
	}

	if viper.GetString("log.format") == "text" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	zerolog.SetGlobalLevel(level)
	if level == zerolog.TraceLevel {
		log.Logger = log.With().Caller().Logger()
	}

	log.Logger.Level(level)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Printf("home directory not found %s\n", err.Error())
			os.Exit(1)
		}

		viper.AddConfigPath(".")
		viper.AddConfigPath(home)
		viper.AddConfigPath("/app")
		viper.SetConfigName("geo")
	}

	replacer := strings.NewReplacer("-", "_", ".", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.SetEnvPrefix("GEO")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}

	ctx := context.Background()
	configureLogging(ctx)

	// initialize sentry if a DSN is configured via sentry.dsn in the config
	// file or the GEO_SENTRY_DSN environment variable
	if dsn := viper.GetString("sentry.dsn"); dsn != "" {
		if err := sentry.Init(sentry.ClientOptions{Dsn: dsn}); err != nil {
			log.Warn().Err(err).Msg("failed to initialize Sentry")
		} else {
			// log that sentry error tracking is active
			log.Info().Msg("Sentry error tracking enabled")
		}
	}

	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		ctx := context.Background()
		log.Info().Str("file", e.Name).Msg("reloading config")
		configureLogging(ctx)
		configureCache(ctx)
	})
}
