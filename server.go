package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

func setupServer(config *tomlConfig) *http.Server {
	mux := http.NewServeMux()
	var rootHandler http.Handler

	if verbose {
		rootHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			zap.L().Info("HTTP request", zap.String("addr", r.RemoteAddr), zap.String("method", r.Method), zap.String("path", r.URL.String()))
			mux.ServeHTTP(w, r)
		})
	} else {
		rootHandler = mux
	}

	server := &http.Server{
		Addr:    config.Depot.ListenAddress,
		Handler: rootHandler,
	}

	if config.Depot.RepositoryListing {
		zap.L().Info("Repository listing is enabled")
		mux.HandleFunc("/repository", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)

			fmt.Fprint(w, `<pre>`)
			for repo := range config.Repositories {
				fmt.Fprintf(w, "<a href=\"/repository/%s/\">%s/</a>\n", repo, repo)
			}
			fmt.Fprint(w, `</pre>`)
		})
	}

	for name, repo := range config.Repositories {
		handler, route := repositoryHandler(name, repo)
		mux.Handle(route, handler)

		if verbose {
			zap.L().Info("Mapped route", zap.String("from", repo.Path), zap.String("to", route))
		}
	}

	if config.Depot.APIEnabled {
		zap.L().Info("API endpoint is enabled")
		setupJSONRoute(mux, config)
	}

	return server
}

func repositoryHandler(name string, info repositoryInfo) (http.HandlerFunc, string) {
	repoRoute := fmt.Sprintf("/repository/%s/", name)
	fileServer := http.StripPrefix(repoRoute, http.FileServer(http.Dir(info.Path)))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, credsSupplied := r.BasicAuth()

		// Check user access
		accessRequiresAuth := !info.IsPublic()
		deployRequiresAuth := len(info.DeployCredentials) > 0
		canAccess := !accessRequiresAuth
		canDeploy := !deployRequiresAuth

		if deployRequiresAuth {
			canDeploy = credsSupplied && checkAuthentication(info.DeployCredentials, username, password)
			if canDeploy {
				canAccess = true
			}
		}

		if accessRequiresAuth && !canAccess {
			canAccess = credsSupplied && checkAuthentication(info.Credentials, username, password)
			if !canAccess {
				canDeploy = false
			}
		}

		// Simply serve artifacts
		if r.Method == "GET" || r.Method == "HEAD" {
			if !canAccess {
				w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="Repository %s is protected"`, name))
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
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

			if !canDeploy {
				w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="Repository %s deployment is protected"`, name))
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			file, err := filepath.Rel(repoRoute, r.URL.Path)
			if err != nil {
				panic(err)
			}

			// Open PUT body stream
			requestBody := http.MaxBytesReader(w, r.Body, int64(info.MaxArtifactSize))

			// Set up directories
			filePath := filepath.Join(info.Path, file)
			fileDir := filepath.Dir(filePath)
			if err := os.MkdirAll(fileDir, 0755); err != nil {
				zap.L().Error("failed to create a directory", zap.String("path", fileDir), zap.Error(err))
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}

			// Stream contents to disk
			fileHandle, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				defer func() {
					os.Remove(filePath)
				}()
				zap.L().Error("failed to create file", zap.String("path", fileDir), zap.Error(err))
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}

			defer requestBody.Close()
			defer fileHandle.Close()
			var written int64
			if written, err = io.Copy(fileHandle, requestBody); err != nil {
				defer func() {
					os.Remove(filePath)
				}()
				zap.L().Error("failed to stream PUT body to disk", zap.Error(err))
				if strings.Contains(err.Error(), "request body too large") {
					http.Error(w, "artifact too large", http.StatusBadRequest)
				} else {
					http.Error(w, "internal error", http.StatusInternalServerError)
				}
				return
			}
			zap.L().Debug("file written on disk", zap.String("filePath", filePath), zap.Int64("bytes", written))

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
			return
		}

		http.Error(w, "bad request", http.StatusBadRequest)
	}), repoRoute
}

func checkAuthentication(credentials []string, username string, password string) bool {
	for _, creds := range credentials {
		splitted := strings.Split(creds, ":")
		if len(splitted) != 2 {
			// Invalid credentials :(
			continue
		}
		checkUsername := splitted[0]
		checkPassword := splitted[1]

		if checkUsername == username && checkPassword == password {
			return true
		}
	}
	return false
}

func setupNoCacheHeaders(response http.ResponseWriter) {
	response.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")

	// HTTP/1.0 - https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Pragma
	response.Header().Set("Pragma", "no-cache")
}
