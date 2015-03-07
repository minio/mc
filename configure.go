package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/user"
	"path"

	"github.com/codegangsta/cli"
	"github.com/minio-io/mc/pkg/s3"
)

const (
	mcConfigDir      = ".minio/mc"
	mcConfigFilename = "config.json"
)

type s3Config struct {
	Auth s3.Auth
}
type mcConfig struct {
	Version string
	S3      s3Config
}

func getMcConfigDir() string {
	u, err := user.Current()
	if err != nil {
		log.Fatalf("mc: Unable to obtain user's home directory. \nERROR[%v]\n", err)
	}

	return path.Join(u.HomeDir + "/" + mcConfigDir)
}

func getMcConfigFilename() string {
	return path.Join(getMcConfigDir() + "/" + mcConfigFilename)
}

func getMcConfig() (config *mcConfig, err error) {
	configBytes, err := ioutil.ReadFile(getMcConfigFilename())
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		return nil, err
	}

	if config.S3.Auth.Endpoint == "" {
		config.S3.Auth.Endpoint = "s3.amazonaws.com"
	}

	return config, nil
}

func parseConfigureInput(c *cli.Context) (config *mcConfig, err error) {
	accessKey := c.String("accesskey")
	secretKey := c.String("secretkey")
	endpoint := c.String("endpoint")
	pathstyle := c.Bool("pathstyle")

	config = &mcConfig{
		Version: "0.1.0",
		S3: s3Config{
			Auth: s3.Auth{
				AccessKey:        accessKey,
				SecretAccessKey:  secretKey,
				Endpoint:         endpoint,
				S3ForcePathStyle: pathstyle,
			},
		},
	}

	return config, nil
}

func doConfigure(c *cli.Context) {
	configData, err := parseConfigureInput(c)
	if err != nil {
		log.Fatal(err)
	}

	jsonConfig, err := json.MarshalIndent(configData, "", "\t")
	if err != nil {
		log.Fatal(err)
	}

	err = os.MkdirAll(getMcConfigDir(), 0755)
	if err != nil {
		log.Fatal(err)
	}

	configFile, err := os.OpenFile(getMcConfigFilename(), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	defer configFile.Close()
	if err != nil {
		log.Fatal(err)
	}

	_, err = configFile.Write(jsonConfig)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("\nConfiguration written to ", getMcConfigFilename())
}

func getNewClient(config *mcConfig) (*s3.Client, error) {
	return &s3.Client{
		Auth:      &config.S3.Auth,
		Transport: http.DefaultTransport,
	}, nil
}
