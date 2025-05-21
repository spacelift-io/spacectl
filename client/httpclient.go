package client

import (
	"net/http"
	"time"
)

var httpClient *http.Client = &http.Client{
	Timeout: 60 * time.Second,
}

func GetHTTPClient() *http.Client {
	return httpClient
}
