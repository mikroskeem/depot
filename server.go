package main

import (
	"fmt"
	"net/http"

	"go.uber.org/zap"
)

func bootServer(repositories map[string]repositoryInfo) error {
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    ":5000",
		Handler: mux,
	}

	for name, repo := range repositories {
		fileServer := http.FileServer(http.Dir(repo.Path))
		repoRoute := fmt.Sprintf("/repository/%s/", name)
		mux.Handle(repoRoute, http.StripPrefix(repoRoute, fileServer))

		zap.L().Info("Mapped route", zap.String("from", repo.Path), zap.String("to", repoRoute))
	}

	return server.ListenAndServe()
}
