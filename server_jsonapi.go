package main

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

func setupJSONRoute(mux *http.ServeMux, config *tomlConfig) {
	// Repository listing
	type publicRepositoryInfo struct {
		// Repository name
		Name string `json:"name"`

		// Whether repository is publicly accessible or not
		Public bool `json:"public"`
	}

	mux.HandleFunc("/api/v1/list", func(w http.ResponseWriter, r *http.Request) {
		setupNoCacheHeaders(w)
		w.Header().Set("Content-Type", "application/json")

		var repos []publicRepositoryInfo
		if config.Depot.RepositoryListing {
			w.WriteHeader(http.StatusOK)
			repos = make([]publicRepositoryInfo, 0, len(config.Repositories))
			for n, info := range config.Repositories {
				repos = append(repos, publicRepositoryInfo{
					Name:   n,
					Public: info.IsPublic(),
				})
			}
		} else {
			w.WriteHeader(http.StatusForbidden)
			repos = make([]publicRepositoryInfo, 0)
		}

		if err := json.NewEncoder(w).Encode(repos); err != nil {
			zap.L().Warn("failed to send response", zap.String("url", r.URL.Path), zap.Error(err))
		}
	})
}
