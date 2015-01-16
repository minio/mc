package main

import (
	"log"
	"strings"

	"github.com/codegangsta/cli"
)

type MinioClient struct {
	bucketName string
	keyName    string
	body       string
	bucketAcls string
	policy     string
	region     string
	query      string // TODO
}

var Options = []cli.Command{
	Cp,
	Ls,
	Mb,
	Mv,
	Rb,
	Rm,
	Sync,
	GetObject,
	PutObject,
	ListObjects,
	ListBuckets,
	Configure,
}

func parseInput(c *cli.Context) string {
	var commandName string
	switch len(c.Args()) {
	case 1:
		commandName = c.Args()[0]
	default:
		log.Fatal("command name must not be blank\n")
	}

	var inputOptions []string
	if c.String("bucket") != "" {
		inputOptions = strings.Split(c.String("options"), ",")
	}

	if inputOptions[0] == "" {
		log.Fatal("options cannot be empty with a command name")
	}
	return commandName
}
