package aur

import (
	"errors"
	"io/ioutil"
	"net/http"
)

// DownloadPackageData downloads package data file from AUR; decompression happens automatically
func DownloadPackageData(address string, lastmod string) ([]byte, string, error) {
	req, err := http.NewRequest("GET", address, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("If-Modified-Since", lastmod)

	r, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	if r.StatusCode == 304 {
		return nil, lastmod, errors.New("not modified")
	}
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, "", err
	}

	return body, r.Header.Get("Last-Modified"), nil
}
