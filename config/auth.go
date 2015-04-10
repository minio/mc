package config

type Auth struct {
	AccessKeyID     string
	SecretAccessKey string
}

// IsValidSecretKey - validate secret key
func (a Auth) IsValidSecretKey() bool {
	if len(a.SecretAccessKey) != 40 {
		return false
	}
	return true
}

// IsValidAccessKey - validate access key
func (a Auth) IsValidAccessKey() bool {
	if len(a.AccessKeyID) != 20 {
		return false
	}
	// Is alphanumeric?
	isalnum := func(c rune) bool {
		return '0' <= c && c <= '9' || 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z'
	}
	for _, char := range a.AccessKeyID {
		if isalnum(char) {
			continue
		}
		switch char {
		case '-':
		case '.':
		case '_':
		case '~':
			continue
		default:
			return false
		}
	}
	return true
}
