package config

import "fmt"

// AliasExists - alias exists
type AliasExists struct {
	Name string
}

func (e AliasExists) Error() string {
	return fmt.Sprintf("alias: %s exists", e.Name)
}

// AliasNotFound - alias not found
type AliasNotFound struct {
	Name string
}

func (e AliasNotFound) Error() string {
	return fmt.Sprintf("alias: %s exists", e.Name)
}

// InvalidAuthKeys - invalid authorization keys
type InvalidAuthKeys struct{}

func (e InvalidAuthKeys) Error() string {
	return fmt.Sprintf("invalid authorization key")
}

// HostExists - host exists
type HostExists struct {
	Name string
}

func (e HostExists) Error() string {
	return fmt.Sprintf("host: %s exists", e.Name)
}

// InvalidArgument - invalid argument
type InvalidArgument struct{}

func (e InvalidArgument) Error() string {
	return fmt.Sprintf("invalid argument")
}
