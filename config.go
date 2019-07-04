package main

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

	// Whether JSON REST API queries are allowed or not
	APIEnabled bool `toml:"api_enabled"`
}

type repositoryInfo struct {
	// Path specifies the repository location on filesystem
	Path string `toml:"path"`

	// Credentials are used for generic repository access authentication. If empty, then repository
	// can be accessed freely by anyone
	// Note that these credentials do not grant deployment access.
	Credentials []string `toml:"credentials"`

	// Deploy configures whether deployment to said repository is allowed or not
	Deploy bool `toml:"deploy"`

	// DeployCredentials are used to authenticate deployments.
	// These credentials grant both access and deployment
	DeployCredentials []string `toml:"deploy_credentials"`

	// MaxArtifactSize defines maximum deployable file size in bytes. By default it's 32 megabytes
	MaxArtifactSize uint64 `toml:"max_artifact_size"`
}
