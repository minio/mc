package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"encoding/json"
	"io/ioutil"
	"net/url"
	"os/user"

	"github.com/codegangsta/cli"
	"github.com/minio-io/mc/pkg/s3"
)

const (
	mcConfigDir      = ".minio/mc"
	mcConfigFilename = "config.json"
)

type hostConfig struct {
	Auth s3.Auth
}

type mcConfig struct {
	Version     uint
	DefaultHost string
	Hosts       map[string]hostConfig
	Aliases     map[string]string
}

const (
	currentConfigVersion = 1
)

// Global config data loaded from json config file durlng init(). This variable should only
// be accessed via getMcConfig()
var _config *mcConfig

func getMcConfigDir() string {
	u, err := user.Current()
	if err != nil {
		msg := fmt.Sprintf("Unable to obtain user's home directory. \nError: %s", err)
		fatal(msg)
	}

	return path.Join(u.HomeDir, mcConfigDir)
}

func getMcConfigFilename() string {
	return path.Join(getMcConfigDir(), mcConfigFilename)
}

func getMcConfig() (cfg *mcConfig, err error) {
	if _config != nil {
		return _config, nil
	}

	_config, err = loadMcConfig()
	if err != nil {
		return nil, err
	}

	return _config, nil
}

// chechMcConfig checks for errors in config file
func checkMcConfig(config *mcConfig) (err error) {
	// check for version
	switch {
	case (config.Version != currentConfigVersion):
		return fmt.Errorf("Unsupported version [%d]. Current operating version is [%d]",
			config.Version, currentConfigVersion)

	case len(config.Hosts) > 1:
		for host, hostCfg := range config.Hosts {
			if host == "" {
				return fmt.Errorf("Empty host URL")
			}
			if hostCfg.Auth.AccessKeyID == "" {
				return fmt.Errorf("AccessKeyID is empty for Host [%s]", host)
			}
			if hostCfg.Auth.SecretAccessKey == "" {
				return fmt.Errorf("SecretAccessKey is empty for Host [%s]", host)
			}
		}
	case len(config.Aliases) > 0:
		for aliasName, aliasURL := range config.Aliases {
			_, err := url.Parse(aliasURL)
			if err != nil {
				return fmt.Errorf("Unable to parse URL [%s] for alias [%s]",
					aliasURL, aliasName)
			}
			if !isValidAliasName(aliasName) {
				return fmt.Errorf("Not a valid alias name [%s]. Valid examples are: Area51, Grand-Nagus..",
					aliasName)
			}
		}
	}
	return nil
}

// loadMcConfig decodes json configuration file to mcConfig structure
func loadMcConfig() (config *mcConfig, err error) {
	configFile := getMcConfigFilename()
	_, err = os.Stat(configFile)
	if err != nil {
		return nil, err
	}

	configBytes, err := ioutil.ReadFile(configFile)
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
	accessKeyID := c.String("accesskeyid")
	secretAccesskey := c.String("secretkey")

	if accessKeyID == "" {
		accessKeyID = "YOUR-ACCESS-KEY-ID-HERE"
	}

	if secretAccesskey == "" {
		secretAccesskey = "YOUR-SECRET-ACCESS-KEY-HERE"
	}

	alias := strings.Fields(c.String("alias"))
	if len(alias) == 0 {
		// valid case throw help
		return nil, nil
	}
	if len(alias) != 2 {
		return nil, errors.New("invalid number of arguments for --alias, requires exact 2")
	}
	aliasName := alias[0]
	url := alias[1]
	if strings.HasPrefix(aliasName, "http") {
		return nil, errors.New("invalid alias cannot use http{s}")
	}
	if !strings.HasPrefix(url, "http") {
		return nil, errors.New("invalid url type only supports http{s}")
	}

	config = &mcConfig{
		Version:     currentConfigVersion,
		DefaultHost: "https://s3.amazonaws.com",
		Hosts: map[string]hostConfig{
			"https://s3.amazonaws.com": {
				Auth: s3.Auth{
					AccessKeyID:     accessKeyID,
					SecretAccessKey: secretAccesskey,
				}},
		},
		Aliases: map[string]string{
			"s3":        "https://s3.amazonaws.com/",
			"localhost": "http://localhost:9000/",
			aliasName:   url,
		},
	}
	//config.Aliases[aliasName] = url
	return config, nil
}

// getHostConfig retrieves host specific configuration such as access keys, certs.
func getHostConfig(host string) (hostCfg *hostConfig, err error) {
	_, err = url.Parse(host)
	if err != nil {
		return nil, err

	}
	config, err := getMcConfig()
	hostCfg.Auth.AccessKeyID = config.Hosts[host].Auth.AccessKeyID
	hostCfg.Auth.SecretAccessKey = config.Hosts[host].Auth.SecretAccessKey
	return hostCfg, nil
}

// saveConfig writes configuration data in json format to config file.
func saveConfig(c *cli.Context) error {
	configData, err := parseConfigInput(c)
	if err != nil {
		return err
	}

	jsonConfig, err := json.MarshalIndent(configData, "", "\t")
	if err != nil {
		return err
	}

	err = os.MkdirAll(getMcConfigDir(), 0755)
	if !os.IsExist(err) && err != nil {
		return err
	}

	configFile, err := os.OpenFile(getMcConfigFilename(), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer configFile.Close()

	_, err = configFile.Write(jsonConfig)
	if err != nil {
		return err
	}
	return nil
}

// doConfigCmd is the handler for "mc config" sub-command.
func doConfigCmd(c *cli.Context) {
	switch true {
	case c.Bool("completion") == true:
		getBashCompletion()
	default:
		err := saveConfig(c)
		if os.IsExist(err) {
			log.Fatalf("mc: Please rename your current configuration file [%s]\n", getMcConfigFilename())
		}

		if err != nil {
			log.Fatalf("mc: Unable to generate config file [%s]. \nError: %v\n", getMcConfigFilename(), err)
		}
		info("Configuration written to " + getMcConfigFilename() + "\n")

	}
}
