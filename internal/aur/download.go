package aur

import (
	"io/ioutil"
	"net/http"
)

// Download package data file from AUR; decompression happens automatically
func DownloadPackageData(address string) ([]byte, error) {
	r, err := http.Get(address)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}
