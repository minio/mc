package client

import (
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/minio-io/minio/pkg/iodine"
)

// URLType - enum of different url types
type URLType int

// enum types
const (
	URLUnknown    URLType = iota // Unknown type
	URLObject                    // Minio and S3 compatible object storage
	URLFilesystem                // POSIX compatible file systems
)

// GuessPossibleURL - provide guesses for possible mistakes in user input url
func GuessPossibleURL(urlStr string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}
	if u.Scheme == "file" || !strings.Contains(urlStr, ":///") {
		possibleURL := u.Scheme + ":///" + u.Host + u.Path
		guess := fmt.Sprintf("Did you mean? %s", possibleURL)
		return guess
	}
	// TODO(y4m4) - add more guesses if possible
	return ""
}

// GetURLType returns the type of URL.
func GetURLType(urlStr string) URLType {
	u, err := url.Parse(urlStr)
	if err != nil {
		return URLUnknown
	}

	if u.Scheme == "http" || u.Scheme == "https" {
		return URLObject
	}

	// while Scheme file, host should be empty
	if u.Scheme == "file" && u.Host == "" && strings.Contains(urlStr, ":///") {
		return URLFilesystem
	}

	// MS Windows OS: Match drive letters
	if runtime.GOOS == "windows" {
		if regexp.MustCompile(`^[a-zA-Z]?$`).MatchString(u.Scheme) {
			return URLFilesystem
		}
	}

	// local path, without the file:///
	if u.Scheme == "" {
		return URLFilesystem
	}

	return URLUnknown
}

// URL2Object converts URL to bucket and objectname
func URL2Object(urlStr string) (bucketName, objectName string, err error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return "", "", iodine.New(err, nil)
	}

	switch GetURLType(urlStr) {
	case URLFilesystem:
		if runtime.GOOS == "windows" {
			bucketName, objectName = filepath.Split(u.String())
		} else {
			bucketName, objectName = filepath.Split(u.Path)
		}
	default:
		splits := strings.SplitN(u.Path, "/", 3)
		switch len(splits) {
		case 0, 1:
			bucketName = ""
			objectName = ""
		case 2:
			bucketName = splits[1]
			objectName = ""
		case 3:
			bucketName = splits[1]
			objectName = splits[2]
		}
	}
	return bucketName, objectName, nil
}
