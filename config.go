package main

import (
	"io"

	"github.com/BurntSushi/toml"
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

	// Whether JSON REST API queries are allowed or not
	APIEnabled bool `toml:"api_enabled"`

	// Whether to save configuration changes done on runtime on Depot exit or not
	SaveConfigChanges bool `toml:"save_config_changes"`
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

// Validates configuration
func (t *tomlConfig) Validate() error {
	// Validate listen address
	if len(t.Depot.ListenAddress) == 0 {
		t.Depot.ListenAddress = ":5000"
	}

	ensureCopy := func(m **repositoryInfo, source *repositoryInfo) {
		if *m == nil {
			*m = &repositoryInfo{}
			**m = *source
		}
	}

	// Validate repository information
	for n, info := range t.Repositories {
		modified := (*repositoryInfo)(nil)

		// Need to copy structs here, maps don't work like I expected :(
		if info.MaxArtifactSize == 0 {
			ensureCopy(&modified, &info)
			modified.MaxArtifactSize = 32 << 20
		}

		// Work around toml library not encoding nil arrays
		if info.Credentials == nil || len(info.Credentials) == 0 {
			ensureCopy(&modified, &info)
			modified.Credentials = []string{}
		}

		if info.DeployCredentials == nil || len(info.DeployCredentials) == 0 {
			ensureCopy(&modified, &info)
			modified.DeployCredentials = []string{}
		}

		// If entry is modified, replace
		if modified != nil {
			t.Repositories[n] = *modified
		}
	}

	return nil
}

// Dumps configuration
func (t *tomlConfig) Dump(writer io.Writer) error {
	w := toml.NewEncoder(writer)
	w.Indent = "    "
	return w.Encode(t)
}

// Returns whether given repository is public or not
func (i *repositoryInfo) IsPublic() bool {
	return len(i.Credentials) == 0
}
