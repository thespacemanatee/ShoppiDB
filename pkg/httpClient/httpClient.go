package httpClient

import (
	"net/http"
	"time"
)

func GetHTTPClient() *http.Client {
	tr := &http.Transport{
		MaxIdleConns:       100,
		IdleConnTimeout:    30 * time.Second,
		MaxConnsPerHost:    100,
		DisableCompression: true,
	}
	client := &http.Client{
		Timeout:   300 * time.Millisecond,
		Transport: tr,
	}
	return client
}
