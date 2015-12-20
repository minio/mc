package main

import (
	"regexp"
	"strings"

	"github.com/minio/mc/pkg/client"
)

// isValidSecretKey - validate secret key.
func isValidSecretKey(secretKey string) bool {
	if secretKey == "" {
		return true
	}
	regex := regexp.MustCompile("^.{40}$")
	return regex.MatchString(secretKey)
}

// isValidAccessKey - validate access key.
func isValidAccessKey(accessKey string) bool {
	if accessKey == "" {
		return true
	}
	regex := regexp.MustCompile("^[A-Z0-9\\-\\.\\_\\~]{20}$")
	return regex.MatchString(accessKey)
}

// isValidHostURL - validate input host url.
func isValidHostURL(hostURL string) bool {
	if strings.TrimSpace(hostURL) == "" {
		return false
	}
	url := client.NewURL(hostURL)
	if url.Scheme != "https" && url.Scheme != "http" {
		return false
	}
	if url.Path != "" && url.Path != "/" {
		return false
	}
	return true
}
