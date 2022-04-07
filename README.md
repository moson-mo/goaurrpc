# goaurrpc
[![GitHub Workflow Status](https://img.shields.io/github/workflow/status/moson-mo/goaurrpc/Go)](https://github.com/moson-mo/goaurrpc/actions) [![Coverage](https://img.shields.io/badge/Coverage-95.5%25-brightgreen)](https://github.com/moson-mo/goaurrpc/blob/main/test_coverage.out) [![Go Report Card](https://goreportcard.com/badge/github.com/moson-mo/goaurrpc)](https://goreportcard.com/report/github.com/moson-mo/goaurrpc)
### An implementation of the [aurweb](https://gitlab.archlinux.org/archlinux/aurweb) (v6) - /rpc - REST API service in go

This project implements the /rpc endpoints (REST API) as described [here](https://aur.archlinux.org/rpc/), as well as the "suggest" type.  

Main goal is to increase the performance.  

### Areas of improvement

In the current version (aurweb v6.0.25 was used for comparison/benchmarking), the bottleneck seems to be the database access.  
When a client makes a request an SQL statement is generated and a query is being run against the mariadb server.  
A pretty normal scenario in web application.

### Thoughts

##### In-Memory DB

Since the /rpc endpoint only returns data and does not need to perform any CUD operations, we could hold the package data in memory as well.

Like in the following scenario:

* Fetch all the necessary data from the database in keep it in memory.
* Assemble the data that is being returned to the client from the in-memory data.
* Periodically refresh the data

Doing that you don't need to bother about expensive database queries to your RDBMS.  
You'd run one big query every now and then to get fresh data...

Now for this POC we'll be using package data that is being composed by aurweb on a regular basis.  
Basically it's dumping package data into a gzip compressed JSON file.  
Turns out that this file does contain all the necessary information that is being exposed by the /rpc API. Nice!

### Approach

We periodically fetch the file (`packages-meta-ext-v1.json.gz`) with the package data and load it into memory.  
Instead of fetching data from the DB for each we request we utilize the in-memory data to compose our result and return it to the client.  
(In a "real" scenario you'd load this data directly from the DB; Does not matter for our performance tests though)

### Setup

For the sake of benchmarking I've installed aurweb on a bare metal system according to [these instructions](https://gitlab.archlinux.org/archlinux/aurweb/-/blob/master/TESTING)  
The machine is equipped with a 4 core Intel Core i5-4590T CPU, 16 GB RAM and and SSD. Pretty low-spec nowadays.  
Redis is used for caching (does not really make any difference. It seems in terms of the API endpoint it's only used to cache the rate-limit data)

The package data was generated by the `gendummydata.py` script, configured with 80K packages and 95K users.  
That pretty much matches the current numbers in the AUR.  
A small modification has been made to also generate descriptions for the packages otherwise they'd be empty in the DB (which has some effect on certain /rpc requests)  
This data was exported and used for the comparison, so that both, the FastAPI and goaurrpc have the same basis and can be compared properly.

The cron docker container for the FastAPI solution has been completely disabled so that no export or maintenance jobs are running while we do the benchmarking.

The benchmarks are being run from a Zen2 8-core notebook. The connection between the machines is 1 GBit/s Ethernet.

### Benchmarks

Benchmarks were performed with the Apache Benchmark tool with a total of 1000 requests, running 10 threads in parallel.
5 worker threads were used for FastAPI.  
During the tests, CPU consumption on the host was pretty close to the maximum for both solutions (> 350% usage, 4-cores)  

#### Results

* "suggest" lookup: **~17x** faster (**8345.16** requests per seconds vs. **487.16** r/s)
* "info" lookup: **~35x** faster (**9326.27** r/s vs. **268.50** r/s)
* "search" lookup: **~52x** faster (**853.02** r/s vs. **16.35** r/s) 

Now these benchmarks have been performed on a pretty low-spec machine.  
Doing the same on a more performing machine should show even better results.   

##### Type "suggest"

- FastAPI

```
Server Software:        uvicorn
Server Hostname:        192.168.0.11
Server Port:            18000

Document Path:          /rpc?v=5&type=suggest&arg=attest
Document Length:        72 bytes

Concurrency Level:      10
Time taken for tests:   2.053 seconds
Complete requests:      1000
Failed requests:        0
Total transferred:      466000 bytes
HTML transferred:       72000 bytes
Requests per second:    487.16 [#/sec] (mean)
Time per request:       20.527 [ms] (mean)
Time per request:       2.053 [ms] (mean, across all concurrent requests)
Transfer rate:          221.70 [Kbytes/sec] received
```

- goaurrpc

```
Server Software:        
Server Hostname:        192.168.0.11
Server Port:            10666

Document Path:          /rpc?v=5&type=suggest&arg=attest
Document Length:        72 bytes

Concurrency Level:      10
Time taken for tests:   0.120 seconds
Complete requests:      1000
Failed requests:        0
Total transferred:      180000 bytes
HTML transferred:       72000 bytes
Requests per second:    8345.16 [#/sec] (mean)
Time per request:       1.198 [ms] (mean)
Time per request:       0.120 [ms] (mean, across all concurrent requests)
Transfer rate:          1466.92 [Kbytes/sec] received
```

##### Type "info"

- FastAPI

```
Server Software:        uvicorn
Server Hostname:        192.168.0.11
Server Port:            18000

Document Path:          /rpc?v=5&type=info&arg=attest
Document Length:        804 bytes

Concurrency Level:      10
Time taken for tests:   3.724 seconds
Complete requests:      1000
Failed requests:        0
Total transferred:      1199000 bytes
HTML transferred:       804000 bytes
Requests per second:    268.50 [#/sec] (mean)
Time per request:       37.244 [ms] (mean)
Time per request:       3.724 [ms] (mean, across all concurrent requests)
Transfer rate:          314.39 [Kbytes/sec] received
```

- goaurrpc

```
Server Software:        
Server Hostname:        192.168.0.11
Server Port:            10666

Document Path:          /rpc?v=5&type=info&arg=attest
Document Length:        804 bytes

Concurrency Level:      10
Time taken for tests:   0.107 seconds
Complete requests:      1000
Failed requests:        0
Total transferred:      913000 bytes
HTML transferred:       804000 bytes
Requests per second:    9326.27 [#/sec] (mean)
Time per request:       1.072 [ms] (mean)
Time per request:       0.107 [ms] (mean, across all concurrent requests)
Transfer rate:          8315.32 [Kbytes/sec] received
```

##### Type "search"

- FastAPI

```
Server Software:        uvicorn
Server Hostname:        192.168.0.11
Server Port:            18000

Document Path:          /rpc?v=5&type=search&arg=attest
Document Length:        3211 bytes

Concurrency Level:      10
Time taken for tests:   61.175 seconds
Complete requests:      1000
Failed requests:        0
Total transferred:      3607000 bytes
HTML transferred:       3211000 bytes
Requests per second:    16.35 [#/sec] (mean)
Time per request:       611.748 [ms] (mean)
Time per request:       61.175 [ms] (mean, across all concurrent requests)
Transfer rate:          57.58 [Kbytes/sec] received
```

- goaurrpc

```
Server Software:        
Server Hostname:        192.168.0.11
Server Port:            10666

Document Path:          /rpc?v=5&type=search&arg=attest
Document Length:        3211 bytes

Concurrency Level:      10
Time taken for tests:   1.172 seconds
Complete requests:      1000
Failed requests:        0
Total transferred:      3299000 bytes
HTML transferred:       3211000 bytes
Requests per second:    853.02 [#/sec] (mean)
Time per request:       11.723 [ms] (mean)
Time per request:       1.172 [ms] (mean, across all concurrent requests)
Transfer rate:          2748.17 [Kbytes/sec] received
``` 

### Concerns

- No live data, since the data is being cached in memory and only reloaded every 5 minutes:  

Data could be reloaded more frequently.  
Loading the data from a JSON file or directly from the AUR webserver takes around 2 seconds.
Re-loading data every minute or even in a 10 second interval would be perfectly possible.  
That would only make sense if data is being retrieved directly from the DB though, the JSON file is only exported every 5 minutes...  

- Memory consumption?  

After startup, once all data is loaded the amount of memory that is allocated for the process is ~250 MB  
When data is being re-loaded periodically, the consumption increases temporarily to about ~450 MB   
until the "old set of data" is being garbage-collected.
Sometimes this might take a while. During my tests I have never seen to get bigger than ~500 MB  
(we could forcefully run the GC, but that does not really make sense.)

### How to build

- Download repository `git clone https://github.com/moson-mo/goaurrpc.git`
- `cd goaurrpc`
- Build with: `./build.sh`
- This will create a binary `goaurrpc`

### Config file

See `sample.conf` file. The config file can be loaded by specifying "-c" parameter when running goaurrpc.  
For example: `./goaurrpc -c sample.conf`.
If this parameter is not passed, the default config will be used (sample.conf contains the defaults).  

### Test endpoint

A goaurrpc endpoint can be found here for testing:  
[Test instance](http://server.moson.rocks:10666/rpc)