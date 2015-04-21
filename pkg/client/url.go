package client

import (
	"fmt"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/minio-io/minio/pkg/iodine"
)

// Type - enum of different url types
type Type int

// enum types
const (
	Unknown    Type = iota // Unknown type
	Object                 // Minio and S3 compatible object storage
	Filesystem             // POSIX compatible file systems
)

// GuessPossibleURL - provide guesses for possible mistakes in user input url
func GuessPossibleURL(urlStr string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}
	if u.Scheme == "file" || !strings.Contains(urlStr, ":///") {
		possible := u.Scheme + ":///" + u.Host + u.Path
		guess := fmt.Sprintf("Did you mean? %s", possible)
		return guess
	}
	// TODO(y4m4) - add more guesses if possible
	return ""
}

// GetFilesystemAbsURL - construct an absolute path for all POSIX paths
func GetFilesystemAbsURL(urlStr string) (string, error) {
	var absStrURL string
	var err error

	u, err := url.Parse(urlStr)
	if err != nil {
		return "", iodine.New(err, nil)
	}

	switch true {
	case u.Scheme == "file" && u.IsAbs():
		absStrURL, err = filepath.Abs(filepath.Clean(u.Path))
		if err != nil {
			return "", iodine.New(err, nil)
		}
	default:
		absStrURL, err = filepath.Abs(filepath.Clean(u.String()))
		if err != nil {
			return "", iodine.New(err, nil)
		}
		// url parse converts "\" on windows as "%5c" unescape it
		unescapedAbsURL, err := url.QueryUnescape(absStrURL)
		if err != nil {
			return "", iodine.New(err, nil)
		}
		absStrURL = unescapedAbsURL
	}

	return absStrURL, nil
}

// GetType returns the type of .
func GetType(urlStr string) Type {
	u, err := url.Parse(urlStr)
	if err != nil {
		return Unknown
	}

	if u.Scheme == "http" || u.Scheme == "https" {
		return Object
	}

	// while Scheme file, host should be empty
	// if windows skip this check, not going to support file:/// style on windows
	// we should just check for VolumeName on windows
	if runtime.GOOS != "windows" {
		if u.Scheme == "file" && u.Host == "" && strings.Contains(urlStr, ":///") {
			return Filesystem
		}
	} else {
		if filepath.VolumeName(urlStr) != "" {
			return Filesystem
		}
	}

	// local path, without the file:/// or C:\
	if u.Scheme == "" {
		return Filesystem
	}

	return Unknown
}
