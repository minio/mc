package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"

	"encoding/json"
	"io/ioutil"
	"net/url"
	"os/user"
	"path/filepath"

	"github.com/minio-io/cli"
	"github.com/minio-io/minio/pkg/iodine"
	"github.com/minio-io/minio/pkg/utils/log"
)

const (
	mcConfigDir        = ".mc/"
	mcConfigWindowsDir = "mc/"
	configFile         = "config.json"
)

type auth struct {
	AccessKeyID     string
	SecretAccessKey string
}

type hostConfig struct {
	Auth auth
}

type mcConfig struct {
	Version   uint
	MCVersion string
	Hosts     map[string]hostConfig
	Aliases   map[string]string
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
		log.Debug.Println(iodine.New(err, nil))
		msg := fmt.Sprintf("Unable to obtain user's home directory. \nError: %s", err)
		fatal(msg)
	}
	var p string
	// For windows the path is slightly differently
	if runtime.GOOS == "windows" {
		p = path.Join(u.HomeDir, mcConfigWindowsDir)
	} else {
		p = path.Join(u.HomeDir, mcConfigDir)
	}
	// create the directory if it doesn't exist
	os.MkdirAll(p, 0700)
	return p
}

func getMcConfigFilename() string {
	return path.Join(getMcConfigDir(), configFile)
}

// getMcConfig returns the config data from file. Subsequent calls are
// cached in a private global variable
func getMcConfig() (cfg *mcConfig, err error) {
	if _config != nil {
		return _config, nil
	}

	_config, err = loadMcConfig()
	if err != nil {
		return nil, iodine.New(err, nil)
	}

	return _config, nil
}

// getMcConfig returns the config data from file. Subsequent calls are
// cached in a private global variable
func isMcConfigExist() bool {
	configFile := getMcConfigFilename()
	_, err := os.Stat(configFile)
	if err != nil {
		return false
	}
	return true
}

// chechMcConfig checks for errors in config file
func checkMcConfig(config *mcConfig) (err error) {
	// check for version
	switch {
	case (config.Version != currentConfigVersion):
		err := fmt.Errorf("Unsupported version [%d]. Current operating version is [%d]", config.Version, currentConfigVersion)
		return iodine.New(err, nil)

	case len(config.Hosts) > 1:
		for host, hostCfg := range config.Hosts {
			if host == "" {
				return iodine.New(fmt.Errorf("Empty host URL"), nil)
			}
			if hostCfg.Auth.AccessKeyID == "" {
				return iodine.New(fmt.Errorf("AccessKeyID is empty for Host [%s]", host), nil)
			}
			if hostCfg.Auth.SecretAccessKey == "" {
				return iodine.New(fmt.Errorf("SecretAccessKey is empty for Host [%s]", host), nil)
			}
		}
	case len(config.Aliases) > 0:
		for aliasName, aliasURL := range config.Aliases {
			_, err := url.Parse(aliasURL)
			if err != nil {
				return iodine.New(fmt.Errorf("Unable to parse URL [%s] for alias [%s]", aliasURL, aliasName), nil)
			}
			if !isValidAliasName(aliasName) {
				return iodine.New(fmt.Errorf("Not a valid alias name [%s]. Valid examples are: Area51, Grand-Nagus..", aliasName), nil)
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
		return nil, iodine.New(err, nil)
	}

	configBytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, iodine.New(err, nil)
	}

	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		return nil, iodine.New(err, nil)
	}

	return config, nil
}

// saveConfig writes configuration data in json format to config file.
func saveConfig(ctx *cli.Context) error {
	configData, err := parseConfigInput(ctx)
	if err != nil {
		return iodine.New(err, nil)
	}

	jsonConfig, err := json.MarshalIndent(configData, "", "\t")
	if err != nil {
		return iodine.New(err, nil)
	}

	err = os.MkdirAll(getMcConfigDir(), 0755)
	if !os.IsExist(err) && err != nil {
		return iodine.New(err, nil)
	}

	configFile, err := os.OpenFile(getMcConfigFilename(), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return iodine.New(err, nil)
	}

	_, err = configFile.Write(jsonConfig)
	if err != nil {
		configFile.Close()
		return iodine.New(err, nil)
	}

	configFile.Close()

	// Invalidate cached config
	_config = nil

	// Reload and cache new config
	_, err = getMcConfig()
	if os.IsNotExist(iodine.ToError(err)) {
		return iodine.New(err, nil)
	}

	return nil
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
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		fatal(err)
	}
	_, err = fl.Write(b.Bytes())
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		fatal(err)
	}
	msg := "\nConfiguration written to " + f
	msg = msg + "\n\n$ source ${HOME}/.mc/mc.bash_completion\n"
	msg = msg + "$ echo 'source ${HOME}/.mc/mc.bash_completion' >> ${HOME}/.bashrc\n"
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
	switch true {
	case len(alias) == 0:
		config = &mcConfig{
			Version:   currentConfigVersion,
			MCVersion: "0.9",
			Hosts: map[string]hostConfig{
				"http*://s3*.amazonaws.com": {
					Auth: auth{
						AccessKeyID:     accessKeyID,
						SecretAccessKey: secretAccesskey,
					}},
			},
			Aliases: map[string]string{
				"s3":        "https://s3.amazonaws.com",
				"localhost": "http://localhost:9000",
			},
		}
		return config, nil
	case len(alias) == 2:
		aliasName := alias[0]
		url := alias[1]
		if strings.HasPrefix(aliasName, "http") {
			return nil, iodine.New(errors.New("invalid alias cannot use http{s}"), nil)
		}
		if !strings.HasPrefix(url, "http") {
			return nil, iodine.New(errors.New("invalid url type only supports http{s}"), nil)
		}
		config = &mcConfig{
			Version:   currentConfigVersion,
			MCVersion: "0.9",
			Hosts: map[string]hostConfig{
				"http*://s3*.amazonaws.com": {
					Auth: auth{
						AccessKeyID:     accessKeyID,
						SecretAccessKey: secretAccesskey,
					}},
			},
			Aliases: map[string]string{
				"s3":        "https://s3.amazonaws.com",
				"localhost": "http://localhost:9000",
				aliasName:   url,
			},
		}
		return config, nil
	default:
		return nil, iodine.New(errors.New("invalid number of arguments for --alias, requires exact 2"), nil)
	}
}

// getHostConfig retrieves host specific configuration such as access keys, certs.
func getHostConfig(hostURL string) (*hostConfig, error) {
	_, err := url.Parse(hostURL)
	if err != nil {
		return nil, iodine.New(err, nil)

	}

	config, err := getMcConfig()
	if err != nil {
		return nil, iodine.New(err, nil)
	}

	for globURL, cfg := range config.Hosts {
		match, err := filepath.Match(globURL, hostURL)
		if err != nil {
			return nil, fmt.Errorf("Error parsing glob'ed URL while comparing [%s] [%s]", globURL, hostURL)
		}
		if match {
			var hostCfg hostConfig
			hostCfg.Auth.AccessKeyID = cfg.Auth.AccessKeyID
			hostCfg.Auth.SecretAccessKey = cfg.Auth.SecretAccessKey
			return &hostCfg, nil
		}
	}
	return nil, iodine.New(errors.New("No matching host config found"), nil)
}

//getBashCompletionCmd generates bash completion file.
func getBashCompletionCmd() {
	var b bytes.Buffer
	if os.Getenv("SHELL") != "/bin/bash" {
		fatal("Unsupported shell for bash completion detected.. exiting")
	}
	b.WriteString(mcBashCompletion)
	f := getMcBashCompletionFilename()
	fl, err := os.OpenFile(f, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	defer fl.Close()
	if err != nil {
		if globalDebugFlag {
			log.Debug.Println(iodine.New(err, nil))
		}
		fatal(err)
	}
	_, err = fl.Write(b.Bytes())
	if err != nil {
		if globalDebugFlag {
			log.Debug.Println(iodine.New(err, nil))
		}
		fatal(err)
	}
	msg := "\nConfiguration written to " + f
	msg = msg + "\n\n$ source ${HOME}/.mc/mc.bash_completion\n"
	msg = msg + "$ echo 'source ${HOME}/.mc/mc.bash_completion' >> ${HOME}/.bashrc"
	info(msg)
}

// saveConfigCmd writes config file to disk
func saveConfigCmd(ctx *cli.Context) {
	err := saveConfig(ctx)
	if os.IsExist(iodine.ToError(err)) {
		if globalDebugFlag {
			log.Debug.Println(iodine.New(err, nil))
		}
		msg := fmt.Sprintf("mc: Configuration file [%s] already exists.", getMcConfigFilename())
		fatal(msg)
	}

	if err != nil {
		if globalDebugFlag {
			log.Debug.Println(iodine.New(err, nil))
		}
		msg := fmt.Sprintf("mc: Unable to generate config file [%s]. \nError: %v.", getMcConfigFilename(), err)
		fatal(msg)
	}
	info("Configuration written to " + getMcConfigFilename() + ". Please update your access credentials.")
}

// doConfigCmd is the handler for "mc config" sub-command.
func doConfigCmd(ctx *cli.Context) {
	switch true {
	case ctx.Bool("completion") == true:
		getBashCompletionCmd()
	default:
		saveConfigCmd(ctx)
	}
}
