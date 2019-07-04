package main

import (
	"context"
	"flag"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/BurntSushi/toml"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// Version contains current Depot application version
	// information
	Version    string
	verbose    bool
	configFile string
)

func main() {
	// CLI options
	flag.BoolVar(&verbose, "verbose", false, "Enables verbose logging")
	flag.StringVar(&configFile, "config", "./config.toml", "Configuration file location")
	flag.Parse()

	// Setup signal handling
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	// Setup logging
	defer zap.L().Sync()
	if err := configureLogging(verbose); err != nil {
		panic(err)
	}

	zap.L().Info("Depot", zap.String("version", Version))
	zap.L().Info("Work in progress, it's far from being done :(")

	// Read configuration
	var rawConfig []byte
	rawConfig, err := ioutil.ReadFile(configFile)
	if err != nil {
		panic(err)
	}

	var config tomlConfig
	if err := toml.Unmarshal(rawConfig, &config); err != nil {
		panic(err)
	}

	if err := config.Validate(); err != nil {
		panic(err)
	}

	// Boot up the HTTP server
	server := setupServer(config.Depot.ListenAddress, config.Depot.RepositoryListing, config.Depot.APIEnabled, config.Repositories)
	go func() {
		zap.L().Info("Starting HTTP server", zap.String("address", config.Depot.ListenAddress))
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			zap.L().Error("Failed to serve", zap.Error(err))
		}
	}()

	// Wait until exit signal
	<-c
	zap.L().Info("Got interrupt signal")

	// Shut down
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	go server.Shutdown(ctx)

	zap.L().Info("Shutting down")
	<-ctx.Done()

	if config.Depot.SaveConfigChanges {
		zap.L().Info("Saving configuration")
		var f *os.File
		if f, err = os.OpenFile(configFile, os.O_CREATE|os.O_WRONLY, 0600); err != nil {
			zap.L().Error("Failed to open config file for saving", zap.String("file", configFile), zap.Error(err))
			goto End
		}
		defer f.Close()

		if err := config.Dump(f); err != nil {
			zap.L().Error("Failed to write configuration file", zap.String("file", configFile), zap.Error(err))
		}
	}

End:
	zap.L().Info("Bye!")
}

func configureLogging(verbose bool) error {
	var cfg zap.Config

	if verbose {
		cfg = zap.NewDevelopmentConfig()
		cfg.Level.SetLevel(zapcore.DebugLevel)
	} else {
		cfg = zap.NewProductionConfig()
		cfg.Level.SetLevel(zapcore.InfoLevel)
	}

	cfg.Encoding = "console"
	cfg.OutputPaths = []string{
		"stdout",
	}

	logger, err := cfg.Build()
	if err != nil {
		return err
	}

	zap.ReplaceGlobals(logger)

	return nil
}
