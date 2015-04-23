package client

import "net/url"

// Type - enum of different url types
type Type int

// enum types
const (
	Unknown    Type = iota // Unknown type
	Object                 // Minio and S3 compatible object storage
	Filesystem             // POSIX compatible file systems
)

// GetType returns the type of URL
func GetType(urlStr string) Type {
	u, err := url.Parse(urlStr)
	if err != nil {
		return Unknown
	}

	if u.Scheme == "http" || u.Scheme == "https" {
		return Object
	}

	return Filesystem
}

// GetTypeToString returns the type of URL as string
func GetTypeToString(t Type) string {
	switch t {
	case Object:
		return "Object"
	case Filesystem:
		return "Filesystem"
	default:
		return "Unknown"
	}
}
