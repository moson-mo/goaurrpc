package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/moson-mo/goaurrpc/internal/config"
	"github.com/moson-mo/goaurrpc/internal/rpc"
)

/*
	version can be overridden during build:
	go build -ldflags="-X 'main.version=v1.0.0'"
*/

var version = "v1.3.0"

func main() {
	var settings *config.Settings

	// args
	cfile := flag.String("c", "", "Config file")
	verbose := flag.Bool("v", false, "Verbose")
	vverbose := flag.Bool("vv", false, "Very verbose")

	flag.Parse()

	if *vverbose {
		*verbose = true
	}

	// set configuration data
	if *cfile == "" {
		settings = config.DefaultSettings()
	} else {
		var err error
		settings, err = config.LoadFromFile(*cfile)
		if err != nil {
			panic("Error loading config file: " + err.Error())
		}
	}

	// construct new server and start listening for requests
	fmt.Printf("goaurrpc %s is starting...\n\n", version)
	s, err := rpc.New(*settings, *verbose, *vverbose, version)
	if err != nil {
		panic(err)
	}
	if err = s.Listen(); err != http.ErrServerClosed {
		fmt.Println(err)
	}
	fmt.Printf("goaurrpc %s stopped.\n", version)
}
