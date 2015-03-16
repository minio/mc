package main

import (
	"bytes"
	"fmt"
	"os"
	"path"

	"encoding/json"
	"io/ioutil"
	"os/user"

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
		msg := fmt.Sprintf("mc: Unable to obtain user's home directory. \nERROR[%v]", err)
		fatal(msg)
	}

	return path.Join(u.HomeDir, mcConfigDir)
}

func getMcConfigFilename() string {
	return path.Join(getMcConfigDir(), mcConfigFilename)
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

	return config, nil
}

// getBashCompletion -
func getBashCompletion() {
	var b bytes.Buffer
	if os.Getenv("SHELL") != "/bin/bash" {
		fatal("Unsupported shell for bash completion detected.. exiting")
	}
	b.WriteString(mcBashCompletion)
	f := getMcBashCompletionFilename()
	fl, err := os.OpenFile(f, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	defer fl.Close()
	_, err = fl.Write(b.Bytes())
	if err != nil {
		fatal(err.Error())
	}
	msg := "\nConfiguration written to " + f
	msg = msg + "\n\n$ source ${HOME}/.minio/mc/mc.bash_completion\n"
	msg = msg + "$ echo 'source ${HOME}/.minio/mc/mc.bash_completion' >> ${HOME}/.bashrc\n"
	info(msg)
}

func parseConfigInput(c *cli.Context) (config *mcConfig, err error) {
	accessKey := c.String("accesskey")
	secretKey := c.String("secretkey")
	config = &mcConfig{
		Version: "0.1.0",
		S3: s3Config{
			Auth: s3.Auth{
				AccessKey:       accessKey,
				SecretAccessKey: secretKey,
			},
		},
	}
	if c.Bool("completion") {
		getBashCompletion()
	}
	return config, nil
}

func doConfig(c *cli.Context) {
	configData, err := parseConfigInput(c)
	if err != nil {
		fatal(err.Error())
	}

	jsonConfig, err := json.MarshalIndent(configData, "", "\t")
	if err != nil {
		fatal(err.Error())
	}

	err = os.MkdirAll(getMcConfigDir(), 0755)
	if err != nil {
		fatal(err.Error())
	}

	configFile, err := os.OpenFile(getMcConfigFilename(), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	defer configFile.Close()
	if err != nil {
		fatal(err.Error())
	}

	_, err = configFile.Write(jsonConfig)
	if err != nil {
		fatal(err.Error())
	}
	msg := "\nConfiguration written to " + getMcConfigFilename()
	info(msg)
}
