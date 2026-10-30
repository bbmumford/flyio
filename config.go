/*
 * Copyright (c) 2026 HSTLES / ORBTR Pty Ltd. All Rights Reserved.
 * Queries: licensing@hstles.com
 */
package flyio

import (
	"errors"
	"fmt"
	"log"
	"strings"

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
	// APITokenSecret names the vault secret that holds the API token when the
	// document leaves APIToken empty. FillVaultSecret reads it; Validate then
	// reads no FLY_API_TOKEN, so the named secret is the only source.
	APITokenSecret string `yaml:"apiTokenSecret" json:"apiTokenSecret"`
}

// ErrVaultSecret reports an API token whose named vault secret could not
// supply it: no reader, a read that failed, or a secret holding nothing.
var ErrVaultSecret = errors.New("flyio: vault secret did not supply the API token")

// FillVaultSecret supplies an empty APIToken from the vault secret
// APITokenSecret names, through read. A token the document carries is kept and
// no secret is read; a document naming no secret is left alone. A named secret
// that cannot supply the token fails closed: read is nil, the read fails, or
// the secret holds only whitespace. The error wraps ErrVaultSecret and names the
// setting and the secret, never a value.
//
// read takes a secret's name and returns its value; an endpoint passes one
// backed by its vault reader, after the document loads. This package reads no
// vault itself.
func (c *Config) FillVaultSecret(read func(name string) (string, error)) error {
	if c == nil {
		return errors.New("flyio: FillVaultSecret needs a config")
	}
	name := strings.TrimSpace(c.FlyIO.APITokenSecret)
	if name == "" || strings.TrimSpace(c.FlyIO.APIToken) != "" {
		return nil
	}
	if read == nil {
		return fmt.Errorf("%w: flyio.apiToken names vault secret %q, and no vault reader is open", ErrVaultSecret, name)
	}
	v, err := read(name)
	if err != nil {
		return fmt.Errorf("%w: flyio.apiToken names vault secret %q, which did not read: %v", ErrVaultSecret, name, err)
	}
	if v = strings.TrimSpace(v); v == "" {
		return fmt.Errorf("%w: flyio.apiToken names vault secret %q, which holds no value", ErrVaultSecret, name)
	}
	c.FlyIO.APIToken = v
	return nil
}

// Validate validates the FlyIO configuration with environment variable fallbacks.
// FLY_API_TOKEN supplies the token only when the document neither carries one
// nor names a vault secret for it.
func (c *Config) Validate() error {
	// APIToken validation with envreq fallback
	if c.FlyIO.APIToken == "" && strings.TrimSpace(c.FlyIO.APITokenSecret) == "" {
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
