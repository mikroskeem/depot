package main

import (
	"encoding/json"
	"net/http"
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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		repos := make([]publicRepositoryInfo, 0, len(config.Repositories))
		for n, info := range config.Repositories {
			repos = append(repos, publicRepositoryInfo{
				Name:   n,
				Public: info.IsPublic(),
			})
		}

		json.NewEncoder(w).Encode(repos)
	})
}
