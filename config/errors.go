package config

import "fmt"

type AliasExists struct {
	Name string
}

func (e AliasExists) Error() string {
	return fmt.Sprintf("alias: %s exists", e.Name)
}

type AliasNotFound struct {
	Name string
}

func (e AliasNotFound) Error() string {
	return fmt.Sprintf("alias: %s exists", e.Name)
}

type InvalidAuthKeys struct{}

func (e InvalidAuthKeys) Error() string {
	return fmt.Sprintf("invalid authorization key")
}

type HostExists struct {
	Name string
}

func (e HostExists) Error() string {
	return fmt.Sprintf("host: %s exists", e.Name)
}

type InvalidArgument struct{}

func (e InvalidArgument) Error() string {
	return fmt.Sprintf("invalid argument")
}
