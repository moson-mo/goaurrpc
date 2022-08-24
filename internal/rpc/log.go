package rpc

import (
	"log"
)

// Log writes a log message either to a file or stdout
func (s *server) Log(a ...any) {
	log.Println(a...)
}

// LogVerbose writes log messages if the verbose flag is set
func (s *server) LogVerbose(a ...any) {
	if s.verbose {
		s.Log(a...)
	}
}
