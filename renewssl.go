/*
 * Copyright (c) 2026 Brandon Miller-Mumford.
 * Licensed under the MIT License; see LICENSE.
 */
// fly/renewssl.go
package flyio

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

const graphqlEndpoint = "https://api.fly.io/graphql"

// httpClient with dial/TLS/header timeouts so nothing hangs indefinitely.
var httpClient = &http.Client{
	Transport: &http.Transport{
		DialContext:           (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	},
}

type gqlRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

type certQuery struct {
	Data struct {
		App struct {
			Certificate struct {
				Issued struct {
					Nodes []struct {
						ExpiresAt time.Time `json:"expiresAt"`
					} `json:"nodes"`
				} `json:"issued"`
			} `json:"certificate"`
		} `json:"app"`
	} `json:"data"`
}

func doGraphQL(ctx context.Context, token string, req gqlRequest, out interface{}) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", graphqlEndpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpReq.Header.Set("Authorization", "token "+token)

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("http do: %w", err)
	}
	// By the time this runs the response is decoded or refused; a failed close
	// of its body changes neither, so the close error is dropped.
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func getExpiry(ctx context.Context, token, app, hostname string) (time.Time, error) {
	// per-call timeout
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req := gqlRequest{
		Query: `
query($app:String!,$hostname:String!){
  app(name:$app){
    certificate(hostname:$hostname){
      issued { nodes { expiresAt } }
    }
  }
}`,
		Variables: map[string]interface{}{"app": app, "hostname": hostname},
	}

	var out certQuery
	if err := doGraphQL(ctx, token, req, &out); err != nil {
		return time.Time{}, fmt.Errorf("fetch expiry %s: %w", hostname, err)
	}
	nodes := out.Data.App.Certificate.Issued.Nodes
	if len(nodes) == 0 {
		return time.Time{}, fmt.Errorf("no certificate for %q", hostname)
	}
	return nodes[0].ExpiresAt, nil
}

func renewCert(ctx context.Context, token, app, hostname string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	req := gqlRequest{
		Query: `
mutation($app:String!,$hostname:String!){
  renewCertificate(app:$app,hostname:$hostname){ success }
}`,
		Variables: map[string]interface{}{"app": app, "hostname": hostname},
	}
	// A GraphQL failure arrives with status 200: the renewal is done only when
	// the reply carries no errors and renewCertificate reports success.
	var reply struct {
		Data struct {
			RenewCertificate *struct {
				Success bool `json:"success"`
			} `json:"renewCertificate"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := doGraphQL(ctx, token, req, &reply); err != nil {
		return fmt.Errorf("renewal for %s: %w", hostname, err)
	}
	if len(reply.Errors) > 0 {
		return fmt.Errorf("renewal for %s: %s", hostname, reply.Errors[0].Message)
	}
	if reply.Data.RenewCertificate == nil || !reply.Data.RenewCertificate.Success {
		return fmt.Errorf("renewal for %s: the API did not report success", hostname)
	}
	return nil
}

// StartAutoRenew loops forever. For each host:
//   - on expiry-fetch or renew error it calls onError(err) and CONTINUES to the next host,
//   - otherwise logs success/failure on stdout.
//
// After all hosts are handled it waits checkPeriod, then repeats.
// Only exits if ctx is cancelled.
// StartAutoRenew begins a periodic certificate renewal loop for the provided
// Fly.io app and hostnames. All configuration (token, app, hostnames) is now
// passed externally instead of being read from environment or inline config.
//
// token: Fly.io API token
// app:   Fly.io app name
// hostnames: slice of hostnames whose certs should be monitored
// onError: callback invoked when an expiry fetch or renewal fails (non-blocking);
// nil logs each failure instead
func StartAutoRenew(ctx context.Context, token, app string, hostnames []string, onError func(error)) {
	if onError == nil {
		onError = func(err error) { log.Printf("SSL auto-renew: %v", err) }
	}
	log.Println("🔄 Starting SSL auto-renew…")
	// normalize hostnames
	hosts := make([]string, 0, len(hostnames))
	for _, h := range hostnames {
		if hh := strings.TrimSpace(h); hh != "" {
			hosts = append(hosts, hh)
		}
	}

	const (
		threshold   = 7 * 24 * time.Hour // renew if <7d
		checkPeriod = 72 * time.Hour     // run every 3d
	)
	ticker := time.NewTicker(checkPeriod)
	defer ticker.Stop()

	log.Printf("App %q hosts: %v (threshold %v, interval %v)",
		app, hosts, threshold, checkPeriod,
	)

	for {
		dbg.Printf("Beginning certificate check cycle")
		for _, raw := range hosts {
			h := strings.TrimSpace(raw)
			dbg.Printf("Checking %s", h)

			exp, err := getExpiry(ctx, token, app, h)
			if err != nil {
				onError(fmt.Errorf("expiry error for %s: %w", h, err))
				continue
			}

			until := time.Until(exp)
			if until < threshold {
				dbg.Printf("%s expires in %v, renewing", h, until)
				if err := renewCert(ctx, token, app, h); err != nil {
					onError(fmt.Errorf("renewal error for %s: %w", h, err))
				} else {
					dbg.Printf("renewed %s", h)
				}
			} else {
				dbg.Printf("%s valid for %v", h, until)
			}
		}

		select {
		case <-ctx.Done():
			log.Println("🔚 Context done; stopping auto-renew")
			return
		case <-ticker.C:
			// next cycle
		}
	}
}
