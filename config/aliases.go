package config

type Aliases map[string]string

func (a Aliases) Set(aliasName, urlName string) {
	a[aliasName] = urlName
}

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
