package main

import (
	"regexp"
	"strings"

	"github.com/minio/mc/pkg/client"
)

// isValidSecretKey - validate secret key.
func isValidSecretKey(secretAccessKey string) bool {
	if secretAccessKey == "" {
		return true
	}
	regex := regexp.MustCompile("^.{40}$")
	return regex.MatchString(secretAccessKey)
}

// isValidAccessKey - validate access key.
func isValidAccessKey(accessKeyID string) bool {
	if accessKeyID == "" {
		return true
	}
	regex := regexp.MustCompile("^[A-Z0-9\\-\\.\\_\\~]{20}$")
	return regex.MatchString(accessKeyID)
}

// isValidKeys validates both access and secret key.
func isValidKeys(accessKeyID, secretAccessKey string) bool {
	if isValidAccessKey(accessKeyID) && isValidSecretKey(secretAccessKey) {
		return true
	}
	return false
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
