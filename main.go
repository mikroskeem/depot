package main

import (
	"flag"
	"io/ioutil"

	"github.com/BurntSushi/toml"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// Version contains current Depot application version
	// information
	Version string
	verbose bool
)

type tomlConfig struct {
	// Depot contains configuration for the application itself
	Depot depotConfig `toml:"depot"`

	// Repositories is a map of repository names to their info
	Repositories map[string]repositoryInfo `toml:"repositories"`
}

type depotConfig struct {
	// An address where HTTP server should listen on
	ListenAddress string `toml:"listen_address"`

	// Whether listing repositories should be allowed or not
	RepositoryListing bool `toml:"repository_listing"`
}

type repositoryInfo struct {
	// Path specifies the repository location on filesystem
	Path        string   `toml:"path"`
	Deploy      bool     `toml:"deploy"`
	Credentials []string `toml:"credentials"`
}

func main() {
	// CLI options
	verbose = *flag.Bool("verbose", false, "Enables verbose logging")
	flag.Parse()

	// Setup logging
	defer zap.L().Sync()
	if err := configureLogging(verbose); err != nil {
		panic(err)
	}

	zap.L().Info("Depot", zap.String("version", Version))
	zap.L().Info("Work in progress, it's far from being done :(")

	// Read configuration
	var rawConfig []byte
	rawConfig, err := ioutil.ReadFile("./config.toml")
	if err != nil {
		panic(err)
	}

	var config tomlConfig
	if err := toml.Unmarshal(rawConfig, &config); err != nil {
		panic(err)
	}

	var listenAddress string
	if len(config.Depot.ListenAddress) > 0 {
		listenAddress = config.Depot.ListenAddress
	} else {
		listenAddress = ":5000"
	}

	// Boot up the HTTP server
	zap.L().Info("Starting HTTP server", zap.String("address", listenAddress))
	if err := bootServer(listenAddress, config.Depot.RepositoryListing, config.Repositories); err != nil {
		panic(err)
	}
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
