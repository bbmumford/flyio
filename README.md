# flyio

The Fly.io adapter for services hosted on Fly:

- **Config**: a typed Fly config document holding the Fly API token, or the name of a vault secret that holds it (`FillVaultSecret` reads it; `Validate` checks the result). `LoadFlyIOConfig` loads the document through cfgload from `FLYIO_CONFIG_URL` (with `FLYIO_CONFIG_TOKEN`) when it is set, and otherwise takes the token from the environment (`FLY_API_TOKEN`).
- **Health**: `HealthHandler` answers Fly's health checks with the build info (from buildinfo) and the Fly machine, region and app. `RegisterHealthRoute` mounts it on a gorilla `mux.Router` or a standard `http.ServeMux`.
- **Certificates**: `StartAutoRenew` asks Fly's GraphQL API for each hostname's certificate expiry and requests renewal when fewer than 7 days remain, checking every 3 days until its context ends.

## Use

```sh
go get github.com/bbmumford/flyio
```

```go
mux := http.NewServeMux()
flyio.RegisterHealthRoute(mux, "/health")

go flyio.StartAutoRenew(ctx, token, "my-app", []string{"api.example.com"}, func(err error) {
	log.Printf("certificate renewal: %v", err)
})
```

flyio's go.mod replaces argus, buildinfo, cfgload and envreq with sibling directories (`../argus` and so on), so a build outside a workspace needs them checked out beside it.

## Consumers

ORBTR library (the endpoint spine), the orbtr.io and node.orbtr.io endpoints, io tools, and ai xyz.
