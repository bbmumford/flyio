/*
 * Copyright (c) 2026 HSTLES / ORBTR Pty Ltd. All Rights Reserved.
 * Queries: licensing@hstles.com
 */
package flyio

import (
	"log"

	"github.com/bbmumford/cfgload"
	"github.com/bbmumford/envreq"
)

// Config holds FlyIO configuration
type Config struct {
	FlyIO FlyIOSettings `yaml:"flyio" json:"flyio"`
}

// FlyIOSettings defines FlyIO-specific configuration
type FlyIOSettings struct {
	APIToken string `yaml:"apiToken" json:"apiToken"`
}

// Validate validates the FlyIO configuration with environment variable fallbacks
func (c *Config) Validate() error {
	// APIToken validation with envreq fallback
	if c.FlyIO.APIToken == "" {
		apiTokenResult := envreq.Check(envreq.Requirement{
			Name:        "FLY_API_TOKEN",
			Source:      "fly.config",
			Description: "FlyIO API token for deployment operations",
			Optional:    true,
			Sensitive:   true,
		})
		c.FlyIO.APIToken = apiTokenResult.Value
	}
	return nil
}

var globalFlyIOConfig *Config

// LoadFlyIOConfig loads FlyIO configuration using the standard loader
func LoadFlyIOConfig() *Config {
	if globalFlyIOConfig != nil {
		return globalFlyIOConfig
	}

	// Get config URL and token from environment
	configURL := envreq.Check(envreq.Requirement{
		Name:        "FLYIO_CONFIG_URL",
		Source:      "fly.loader",
		Description: "URL for FlyIO configuration",
		Optional:    true,
	}).Value

	configToken := envreq.Check(envreq.Requirement{
		Name:        "FLYIO_CONFIG_TOKEN",
		Source:      "fly.loader",
		Description: "Authentication token for FlyIO config access",
		Optional:    true,
		Sensitive:   true,
	}).Value

	// If no URL provided, use environment variables only
	if configURL == "" {
		log.Println("FLYIO_CONFIG_URL not provided, using environment variables only")
		globalFlyIOConfig = &Config{}
		if err := globalFlyIOConfig.Validate(); err != nil {
			log.Printf("FlyIO config validation failed: %v", err)
		}
		return globalFlyIOConfig
	}

	// Load config from URL using standard loader
	var cfg Config
	err := cfgload.Load(&cfg, cfgload.LoadOptions{
		URL:       configURL,
		AuthToken: configToken,
		// Environment fallback is handled by Validate() method
		AllowEnvFallback: true,
		EnvLoader: func(target interface{}) error {
			// If URL fails, just use environment variables
			c := target.(*Config)
			return c.Validate()
		},
	})

	if err != nil {
		log.Printf("Failed to load FlyIO config: %v, using environment variables", err)
		globalFlyIOConfig = &Config{}
		if err := globalFlyIOConfig.Validate(); err != nil {
			log.Printf("FlyIO config validation failed: %v", err)
		}
		return globalFlyIOConfig
	}

	globalFlyIOConfig = &cfg
	log.Println("FlyIO configuration loaded successfully")
	return globalFlyIOConfig
}

// GetFlyIOAPIToken returns the FlyIO API token
func GetFlyIOAPIToken() string {
	config := LoadFlyIOConfig()
	return config.FlyIO.APIToken
}

// GetFlyIOConfig returns the loaded FlyIO configuration
func GetFlyIOConfig() *Config {
	return LoadFlyIOConfig()
}
