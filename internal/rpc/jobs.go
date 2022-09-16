package rpc

import (
	"sync"
	"time"

	db "github.com/moson-mo/goaurrpc/internal/memdb"
)

// start go-routines for periodic tasks
func (s *server) startJobs(shutdown chan struct{}, wg *sync.WaitGroup) {
	wg.Add(3)

	// starts a go routine that continuously refreshes the package data
	go func() {
		defer wg.Done()
		for {
			select {
			case <-shutdown:
				s.LogVerbose("Stopping routine: Data refresh")
				return
			case <-time.After(time.Duration(s.settings.RefreshInterval) * time.Second):
				s.LogVerbose("Reloading package data...")
				start := time.Now()
				err := s.reloadData()
				if err != nil {
					if err.Error() == "not modified" {
						s.LogVerbose("Reload skipped. File has not been modified.")
					} else {
						s.LogVerbose("Error reloading data: ", err)
					}
				} else {
					elapsed := time.Since(start)
					s.LogVerbose("Successfully reloaded package data in ", elapsed.Milliseconds(), " ms")
				}
			}
		}
	}()

	// starts a go routine that removes rate limits if older than 24h
	go func() {
		defer wg.Done()
		for {
			select {
			case <-shutdown:
				s.LogVerbose("Stopping routine: Ratelimit cleanup")
				return
			case <-time.After(time.Duration(s.settings.RateLimitCleanupInterval) * time.Second):
				s.cleanupRateLimits()
			}
		}
	}()

	// start go routine that cleans up the search cache
	go func() {
		defer wg.Done()
		for {
			select {
			case <-shutdown:
				s.LogVerbose("Stopping routine: Search-Cache cleanup")
				return
			case <-time.After(time.Duration(s.settings.CacheCleanupInterval) * time.Second):
				s.cleanupSearchCache()
			}
		}
	}()
}

// load data from file/url
func (s *server) reloadData() error {
	/*
		use local file for extensive testing -> ptr, err := db.LoadDbFromFile("packages.json")
		we don't want to stress the aur server
	*/
	var ptr *db.MemoryDB
	var lmod string
	var err error
	if s.settings.LoadFromFile {
		ptr, err = db.LoadDbFromFile(s.settings.AurFileLocation)
		if err != nil {
			return err
		}
	} else {
		ptr, lmod, err = db.LoadDbFromUrl(s.settings.AurFileLocation, s.lastmod)
		if err != nil {
			return err
		}
	}
	s.mut.Lock()
	defer s.mut.Unlock()
	s.memDB = ptr
	s.lastmod = lmod
	s.lastRefresh = time.Now()
	return nil
}

// clean up rate limit cache
func (s *server) cleanupRateLimits() {
	s.mutLimit.Lock()
	defer s.mutLimit.Unlock()
	t := time.Now()
	for ip, rl := range s.rateLimits {
		if t.Sub(rl.WindowStart) > time.Duration(s.settings.RateLimitTimeWindow)*time.Second {
			delete(s.rateLimits, ip)
			s.LogVerbose("Removed rate limit for", ip)
		}
	}
}

// clean up search cache
func (s *server) cleanupSearchCache() {
	s.mutCache.Lock()
	defer s.mutCache.Unlock()
	t := time.Now()
	for k, ce := range s.searchCache {
		if t.Sub(ce.TimeAdded) > time.Duration(s.settings.CacheExpirationTime)*time.Second {
			delete(s.searchCache, k)
			s.LogVerbose("Removed cache entry for", k)
		}
	}
}
