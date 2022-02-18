package main

import (
	"flag"
	"fmt"

	"github.com/moson-mo/goaurrpc-poc/internal/config"
	"github.com/moson-mo/goaurrpc-poc/internal/rpc"
)

func main() {
	var settings *config.Settings

	// args
	cfile := flag.String("c", "", "Config file")
	flag.Parse()

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
	s, err := rpc.New(settings)
	if err != nil {
		panic(err)
	}
	fmt.Println(s.Listen())
}
