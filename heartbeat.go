/*
 * Copyright (c) 2026 HSTLES / ORBTR Pty Ltd. All Rights Reserved.
 * Queries: licensing@hstles.com
 */
package flyio

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/bbmumford/buildinfo"
)

// HealthResponse is a minimal health check response for Fly.io monitoring.
// Fly.io health checks primarily need a 2xx status code; additional metadata is optional.
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Machine   string    `json:"machine,omitempty"` // FLY_MACHINE_ID
	Region    string    `json:"region,omitempty"`  // FLY_REGION
	App       string    `json:"app,omitempty"`     // FLY_APP_NAME

	// Build info fields (populated from buildinfo package)
	Version   string `json:"version,omitempty"`
	Commit    string `json:"commit,omitempty"`
	BuildDate string `json:"buildDate,omitempty"`
	BuildID   string `json:"buildID,omitempty"`
	AppName   string `json:"appName,omitempty"`
	Uptime    string `json:"uptime,omitempty"`
	GoVersion string `json:"goVersion,omitempty"`
}

// HealthHandler returns a simple HTTP handler that responds with 200 OK and basic health metadata.
// This satisfies Fly.io's health check requirements (any 2xx status indicates healthy).
// No external dependencies (DB, logging, etc.) are checked to keep it minimal and fast.
func HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bi := buildinfo.Info()
		resp := HealthResponse{
			Status:    "ok",
			Timestamp: time.Now().UTC(),
			Machine:   os.Getenv("FLY_MACHINE_ID"),
			Region:    os.Getenv("FLY_REGION"),
			App:       os.Getenv("FLY_APP_NAME"),
			Version:   bi.Version,
			Commit:    bi.GitCommit,
			BuildDate: bi.BuildDate,
			BuildID:   bi.BuildID,
			AppName:   bi.AppName,
			Uptime:    buildinfo.UptimeString(),
			GoVersion: bi.GoVersion,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

// RegisterHealthRoute registers the health check handler at the given path on the provided router.
// Supports both gorilla mux.Router and stdlib http.ServeMux.
//
// Examples:
//
//	// Gorilla mux
//	router := mux.NewRouter()
//	fly.RegisterHealthRoute(router, "/heartbeat")
//
//	// Stdlib
//	mux := http.NewServeMux()
//	fly.RegisterHealthRoute(mux, "/health")
func RegisterHealthRoute(router interface{}, path string) {
	if path == "" {
		path = "/health"
	}

	// Support any router with Get() (chi, Router interface) or HandleFunc (stdlib)
	switch r := router.(type) {
	case *http.ServeMux:
		r.HandleFunc(path, HealthHandler())
	default:
		if gr, ok := router.(interface {
			Get(string, http.HandlerFunc)
		}); ok {
			gr.Get(path, HealthHandler())
		} else if hf, ok := router.(interface {
			HandleFunc(string, http.HandlerFunc)
		}); ok {
			hf.HandleFunc(path, HealthHandler())
		}
	}
}
