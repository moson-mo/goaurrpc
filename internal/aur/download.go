package aur

import (
	"errors"
	"io"
	"net/http"
	"time"
)

var client = &http.Client{
	Timeout: 10 * time.Second,
}

// DownloadPackageData downloads package data file from AUR; decompression happens automatically
func DownloadPackageData(address string, lastmod time.Time) ([]byte, time.Time, error) {
	req, err := http.NewRequest("GET", address, nil)
	if err != nil {
		return nil, lastmod, err
	}
	req.Header.Set("If-Modified-Since", lastmod.Format(http.TimeFormat))

	r, err := client.Do(req)
	if err != nil {
		return nil, lastmod, err
	}
	defer r.Body.Close()

	if r.StatusCode == 304 {
		io.Copy(io.Discard, r.Body)
		return nil, lastmod, errors.New("not modified")
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, lastmod, err
	}

	newmod, err := http.ParseTime(r.Header.Get("Last-Modified"))
	if err != nil {
		newmod = time.Now()
	}

	return body, newmod, nil
}
