module github.com/bbmumford/flyio

go 1.25.0

require (
	github.com/bbmumford/argus v0.0.0
	github.com/bbmumford/buildinfo v0.0.0
	github.com/bbmumford/cfgload v0.0.0
	github.com/bbmumford/envreq v0.0.0
)

require (
	github.com/kr/text v0.2.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/bbmumford/argus => ../argus

replace github.com/bbmumford/buildinfo => ../buildinfo

replace github.com/bbmumford/cfgload => ../cfgload

replace github.com/bbmumford/envreq => ../envreq
