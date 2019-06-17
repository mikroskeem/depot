package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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

	mux.HandleFunc("/repository", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		for repo := range repositories {
			fmt.Fprintf(w, "<p><a href='/repository/%s/'>%s/</a></p>", repo, repo)
		}
	})

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
		// Do authentication if credentials are configured
		if len(info.Credentials) > 0 {
			username, password, credsSupplied := r.BasicAuth()

			authenticated := false
			if credsSupplied {
				for _, creds := range info.Credentials {
					splitted := strings.Split(creds, ":")
					if len(splitted) != 2 {
						// Invalid credentials :(
						continue
					}
					checkUsername := splitted[0]
					checkPassword := splitted[1]

					if checkUsername == username && checkPassword == password {
						authenticated = true
						break
					}
				}
			}

			if !authenticated {
				w.Header().Set("WWW-Authenticate", fmt.Sprintf("Basic realm=\"Repository %s is protected\"", name))
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}

		// Simply serve artifacts
		if r.Method == "GET" {
			fileServer.ServeHTTP(w, r)
			return
		}

		// Deployment
		if r.Method == "PUT" {
			// Check if deployment is allowed
			if !info.Deploy {
				http.Error(w, "this repository does not allow deployments", http.StatusForbidden)
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
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}

			// Read contents
			defer r.Body.Close()
			contents, err := ioutil.ReadAll(r.Body)
			if err != nil {
				zap.L().Error("failed to read PUT request contents!", zap.Error(err))
				http.Error(w, "bad request", 400)
				return
			}

			// Create file
			if err := ioutil.WriteFile(filePath, contents, 0644); err != nil {
				defer func() {
					os.Remove(filePath)
				}()
				zap.L().Error("failed to create file", zap.String("path", fileDir), zap.Error(err))
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(200)
			w.Write([]byte("ok"))
			return
		}

		http.Error(w, "bad request", 400)
	}), repoRoute
}
