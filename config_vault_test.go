/*
 * Copyright (c) 2026 HSTLES / ORBTR Pty Ltd. All Rights Reserved.
 * Queries: licensing@hstles.com
 */
package flyio

import (
	"errors"
	"strings"
	"testing"

	"github.com/bbmumford/envreq"
)

func TestFillVaultSecretReadsTheNamedSecretOnlyWhenTheTokenIsEmpty(t *testing.T) {
	var read []string
	vault := func(name string) (string, error) {
		read = append(read, name)
		if name == "fly-api-token" {
			return "  vault-token  ", nil
		}
		return "", errors.New("secret not found")
	}
	c := Config{FlyIO: FlyIOSettings{APITokenSecret: "fly-api-token"}}
	if err := c.FillVaultSecret(vault); err != nil || c.FlyIO.APIToken != "vault-token" {
		t.Fatalf("FillVaultSecret = %v, token filled %t", err, c.FlyIO.APIToken == "vault-token")
	}
	kept := Config{FlyIO: FlyIOSettings{APIToken: "doc", APITokenSecret: "fly-api-token"}}
	read = nil
	if err := kept.FillVaultSecret(vault); err != nil || kept.FlyIO.APIToken != "doc" || len(read) != 0 {
		t.Errorf("a document token was not kept without a read: %v, %d reads", err, len(read))
	}
	var none Config
	if err := none.FillVaultSecret(nil); err != nil {
		t.Errorf("a document naming no secret was refused: %v", err)
	}
}

func TestFillVaultSecretFailsClosed(t *testing.T) {
	for name, read := range map[string]func(string) (string, error){
		"no reader": nil,
		"missing":   func(string) (string, error) { return "", errors.New("secret not found") },
		"blank":     func(string) (string, error) { return " ", nil },
	} {
		c := Config{FlyIO: FlyIOSettings{APITokenSecret: "fly-api-token"}}
		err := c.FillVaultSecret(read)
		if !errors.Is(err, ErrVaultSecret) || !strings.Contains(err.Error(), "fly-api-token") ||
			!strings.Contains(err.Error(), "flyio.apiToken") {
			t.Errorf("%s: FillVaultSecret = %v, want ErrVaultSecret naming the setting and the secret", name, err)
		}
	}
}

// A document that names a vault secret makes the vault the only source: Validate
// reads no FLY_API_TOKEN for it.
func TestValidateReadsNoVariableForATokenTheDocumentNamesASecretFor(t *testing.T) {
	envreq.Reset()
	t.Cleanup(envreq.Reset)
	t.Setenv("FLY_API_TOKEN", "env-token")
	c := Config{FlyIO: FlyIOSettings{APITokenSecret: "fly-api-token"}}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	if c.FlyIO.APIToken != "" {
		t.Error("Validate filled the token from FLY_API_TOKEN though the document names a vault secret")
	}
	var plain Config
	if err := plain.Validate(); err != nil || plain.FlyIO.APIToken != "env-token" {
		t.Errorf("a document naming no secret did not take FLY_API_TOKEN: %v", err)
	}
}
