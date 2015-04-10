package config

// Aliases - keep a map
type Aliases map[string]string

// Set - set an alias
func (a Aliases) Set(aliasName, urlName string) {
	a[aliasName] = urlName
}

// Get - get an alias
func (a Aliases) Get(aliasName string) string {
	return a[aliasName]
}

// IsExists verify if alias exists
func (a Aliases) IsExists(aliasName string) bool {
	for alias := range a {
		if alias == aliasName {
			return true
		}
	}
	return false
}
