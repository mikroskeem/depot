package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

func bootServer(repositories map[string]repositoryInfo) error {
	mux := http.NewServeMux()
	server := &http.Server{
		Addr: ":5000",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			zap.L().Info("HTTP request", zap.String("addr", r.RemoteAddr), zap.String("method", r.Method), zap.String("path", r.URL.String()))
			mux.ServeHTTP(w, r)
		}),
	}

	for name, repo := range repositories {
		handler, route := repositoryHandler(name, repo)
		mux.Handle(route, handler)

		zap.L().Info("Mapped route", zap.String("from", repo.Path), zap.String("to", route))
	}

	return server.ListenAndServe()
}

func repositoryHandler(name string, info repositoryInfo) (http.HandlerFunc, string) {
	repoRoute := fmt.Sprintf("/repository/%s/", name)
	fileServer := http.StripPrefix(repoRoute, http.FileServer(http.Dir(info.Path)))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: authentication

		// Simply serve artifacts
		if r.Method == "GET" {
			fileServer.ServeHTTP(w, r)
			return
		}

		// Deployment
		if r.Method == "PUT" {
			// Check if deployment is allowed
			if !info.Deploy {
				w.WriteHeader(403)
				fmt.Fprintf(w, "this repository does not allow deployments")
				return
			}

			file, err := filepath.Rel(repoRoute, r.URL.Path)
			if err != nil {
				panic(err)
			}

			// Set up directories
			filePath := filepath.Join(info.Path, file)
			fileDir := filepath.Dir(filePath)
			if err := os.MkdirAll(fileDir, 0755); err != nil {
				zap.L().Error("failed to create a directory", zap.String("path", fileDir), zap.Error(err))
				w.WriteHeader(500)
				fmt.Fprintf(w, "internal error")
				return
			}

			// Read contents
			defer r.Body.Close()
			contents, err := ioutil.ReadAll(r.Body)
			if err != nil {
				zap.L().Error("failed to read PUT request contents!", zap.Error(err))
				w.WriteHeader(400)
				fmt.Fprintf(w, "bad request")
				return
			}

			// Create file
			if err := ioutil.WriteFile(filePath, contents, 0644); err != nil {
				defer func() {
					os.Remove(filePath)
				}()
				zap.L().Error("failed to create file", zap.String("path", fileDir), zap.Error(err))
				w.WriteHeader(500)
				fmt.Fprintf(w, "internal error")
				return
			}

			w.WriteHeader(200)
			fmt.Fprintf(w, "ok")
			return
		}

		w.WriteHeader(400)
		fmt.Fprintf(w, "bad request")
	}), repoRoute
}
