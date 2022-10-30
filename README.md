# goaurrpc
[![Release](https://img.shields.io/github/v/release/moson-mo/goaurrpc)](https://github.com/moson-mo/goaurrpc/releases) [![GitHub Workflow Status](https://img.shields.io/github/workflow/status/moson-mo/goaurrpc/Go)](https://github.com/moson-mo/goaurrpc/actions) [![Coverage](https://img.shields.io/badge/Coverage-98.2%25-brightgreen)](https://github.com/moson-mo/goaurrpc/blob/main/test_coverage.out) [![Go Report Card](https://goreportcard.com/badge/github.com/moson-mo/goaurrpc)](https://goreportcard.com/report/github.com/moson-mo/goaurrpc)

### An implementation of the [aurweb](https://gitlab.archlinux.org/archlinux/aurweb) (v6) - /rpc - REST API service in go

goaurrpc allows you to run your own self-hosted aurweb /rpc endpoint.  
This project implements the /rpc interface (REST API; version 5) as described [here](https://aur.archlinux.org/rpc/).  

In it's default configuration, package data is being downloaded/refreshed from the AUR every 5 minutes.  
The data is entirely held in-memory as opposed to storing it in a database for example.  
This avoids the need to make heavy database queries for each request.  
For a performance comparison, see [Benchmarks](BENCHMARKS.md)

### How to build

- Download repository `git clone https://github.com/moson-mo/goaurrpc.git`
- `cd goaurrpc`
- Build with: `./build.sh`
- This will create a binary `goaurrpc`

### Config file

See `sample.conf` file. The config file can be loaded by specifying "-c" parameter when running goaurrpc.  
For example: `./goaurrpc -c sample.conf`.
If this parameter is not passed, the default config will be used (sample.conf contains the defaults).  

```
{
	"Port": 10666,
	"AurFileLocation": "https://aur.archlinux.org/packages-meta-ext-v1.json.gz",
	"MaxResults": 5000,
	"RefreshInterval": 300,
	"RateLimit": 4000,
	"LoadFromFile": false,
	"RateLimitCleanupInterval": 600,
	"RateLimitTimeWindow": 86400,
	"TrustedReverseProxies": [
		"127.0.0.1",
		"::1"
	],
	"EnableSSL": false,
	"CertFile": "",
	"KeyFile": "",
	"EnableSearchCache": true,
	"CacheCleanupInterval": 60,
	"CacheExpirationTime": 180,
	"EnableMetrics": true,
	"EnableAdminApi": false,
	"AdminAPIKey": "change-me"
}
```

| Setting | Description |
| ------ | ------ |
| Port | The port number our service is listening on |
| AurFileLocation | Either the URL to the full metadata archive `packages-meta-ext-v1.json.gz` or a local copy of the file |
| MaxResults | The maximum number of package results that are being returned to the client |
| RefreshInterval | The interval (in seconds) in which the metadata file is being reloaded |
| RateLimit | The maximum number of requests that are allowed within the time-window |
| LoadFromFile | Set to true when using a local file instead of a URL for `AurFileLocation` |
| RateLimitCleanupInterval | The interval (in seconds) in which rate-limits are being cleaned up |
| RateLimitTimeWindow | Defines the length of the time window for rate-limiting (in seconds) |
| Trusted reverse proxies | A list of trusted IP-Addresses, in case you use a reverse proxy and need to rely on `X-Real-IP` or `X-Forwarded-For` headers to identify a client (for rate-limiting) |
| EnableSSL | Enables internal SSL/TLS. You'll need to provide `CertFile`and `KeyFile` when enabling it. I'd recommend to use nginx as reverse proxy to add encryption instead |
| CertFile | Path to the cert file (if SSL is enabled) |
| KeyFile | Path to the corresponding key file (if SSL is enabled) |
| EnableSearchCache | Caches data for search queries that have been performed by clients |
| CacheCleanupInterval | The interval (in seconds) for performing cleanup of search-cache entries |
| CacheExpirationTime | The number of seconds an entry should stay in the search-cache |
| EnableMetrics | Enables Prometheus metrics at /metrics |
| EnableAdminApi | Enables the administrative endpoint at /admin |
| AdminAPIKey | The API Key that is to be provided in the header for the /admin endpoint |

### Public endpoint

Feel free to make use of the following public instance of goaurrpc:   

[HTTP](http://server.moson.rocks/rpc) / [HTTPS](https://server.moson.rocks/rpc)

### Future plans / ideas

- Extend request types (see [v6-proposal branch](https://github.com/moson-mo/goaurrpc/tree/v6-proposal))
- Admin REST-API to be able to control goaurrpc at runtime, for example:
  - reload data
  - get statistics (memory consumption, rate limits, etc.)
  - manage rate-limits
  - manage search-cache
- CLI/TUI tool for administration (making use of the admin api)