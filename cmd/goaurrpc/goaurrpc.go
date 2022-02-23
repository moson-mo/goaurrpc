package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/moson-mo/goaurrpc/internal/config"
	"github.com/moson-mo/goaurrpc/internal/rpc"
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
	fmt.Println("Server is starting...")
	s, err := rpc.New(*settings)
	if err != nil {
		panic(err)
	}
	if err = s.Listen(); err != http.ErrServerClosed {
		fmt.Println(err)
	}
	fmt.Println("Server stopped.")
}
