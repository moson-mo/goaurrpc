# goaurrpc
[![Release](https://img.shields.io/github/v/release/moson-mo/goaurrpc)](https://github.com/moson-mo/goaurrpc/releases) [![GitHub Workflow Status](https://img.shields.io/github/workflow/status/moson-mo/goaurrpc/Go)](https://github.com/moson-mo/goaurrpc/actions) [![Coverage](https://img.shields.io/badge/Coverage-95.5%25-brightgreen)](https://github.com/moson-mo/goaurrpc/blob/main/test_coverage.out) [![Go Report Card](https://goreportcard.com/badge/github.com/moson-mo/goaurrpc)](https://goreportcard.com/report/github.com/moson-mo/goaurrpc)

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

### Public endpoint

Feel free to make use of the following public instance of goaurrpc:   

[HTTP](http://server.moson.rocks/rpc) / [HTTPS](https://server.moson.rocks/rpc)