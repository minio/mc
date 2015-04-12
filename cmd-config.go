/*
 * Modern Copy, (C) 2014, 2015 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"bytes"
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
	"github.com/minio-io/mc/pkg/console"
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
	Auth *auth
}

type mcConfig struct {
	Version uint
	Hosts   map[string]hostConfig
	Aliases map[string]string
}

const (
	currentConfigVersion = 1
)

// Global config data loaded from json config file durlng init(). This variable should only
// be accessed via getMcConfig()
var _config *mcConfig

func getMcConfigDir() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", iodine.New(err, nil)
	}
	// For windows the path is slightly differently
	switch runtime.GOOS {
	case "windows":
		return path.Join(u.HomeDir, mcConfigWindowsDir), nil
	default:
		return path.Join(u.HomeDir, mcConfigDir), nil
	}
}

func getOrCreateMcConfigDir() (string, error) {
	p, err := getMcConfigDir()
	if err != nil {
		return "", iodine.New(err, nil)
	}
	err = os.MkdirAll(p, 0700)
	if err != nil {
		return "", iodine.New(err, nil)
	}
	return p, nil
}

func getMcConfigPath() (string, error) {
	dir, err := getMcConfigDir()
	if err != nil {
		return "", iodine.New(err, nil)
	}
	return path.Join(dir, configFile), nil
}

func mustGetMcConfigPath() string {
	p, _ := getMcConfigPath()
	return p
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
	configFile, err := getMcConfigPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(configFile)
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
		return iodine.New(errUnsupportedVersion{old: currentConfigVersion, new: config.Version}, nil)

	case len(config.Hosts) > 1:
		for host, hostCfg := range config.Hosts {
			// don't need to check for availability of AccessKeyID, not having one
			// is a valid case for public buckets
			if host == "" {
				return iodine.New(errEmptyURL{}, nil)
			}
			if hostCfg.Auth == nil {
				return iodine.New(errInvalidAuth{}, nil)
			}
		}
	case len(config.Aliases) > 0:
		for aliasName, aliasURL := range config.Aliases {
			_, err := url.Parse(aliasURL)
			if err != nil {
				return iodine.New(errInvalidAliasURL{alias: aliasName, url: aliasURL}, nil)
			}
			if !isValidAliasName(aliasName) {
				return iodine.New(errInvalidAliasName{alias: aliasName}, nil)
			}
		}
	}
	return nil
}

// loadMcConfig decodes json configuration file to mcConfig structure
func loadMcConfig() (config *mcConfig, err error) {
	configFile, err := getMcConfigPath()
	if err != nil {
		return nil, iodine.New(err, nil)
	}

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

	_, err = getOrCreateMcConfigDir()
	if err != nil {
		return iodine.New(err, nil)
	}

	configPath, err := getMcConfigPath()
	if err != nil {
		return iodine.New(err, nil)
	}

	configFile, err := os.OpenFile(configPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
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
			Version: currentConfigVersion,
			Hosts: map[string]hostConfig{
				"http*://s3*.amazonaws.com": {
					Auth: &auth{
						AccessKeyID:     accessKeyID,
						SecretAccessKey: secretAccesskey,
					}},
				// local minio server can have this empty until we configure it.
				"http*://localhost:*": {
					Auth: &auth{
						AccessKeyID:     "",
						SecretAccessKey: "",
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
			return nil, iodine.New(errInvalidAliasName{alias: aliasName}, nil)
		}
		if !strings.HasPrefix(url, "http") {
			return nil, iodine.New(errInvalidURL{url: url}, nil)
		}
		config = &mcConfig{
			Version: currentConfigVersion,
			Hosts: map[string]hostConfig{
				"http*://s3*.amazonaws.com": {
					Auth: &auth{
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
		return nil, iodine.New(errInvalidArgument{}, nil)
	}
}

// getHostURL -
func getHostURL(u *url.URL) string {
	return u.Scheme + "://" + u.Host
}

// getHostConfig retrieves host specific configuration such as access keys, certs.
func getHostConfig(requestURL string) (*hostConfig, error) {
	u, err := url.Parse(requestURL)
	if err != nil {
		return nil, iodine.New(err, nil)

	}
	config, err := getMcConfig()
	if err != nil {
		return nil, iodine.New(err, nil)
	}

	for globURL, cfg := range config.Hosts {
		match, err := filepath.Match(globURL, getHostURL(u))
		if err != nil {
			return nil, iodine.New(errInvalidGlobURL{glob: globURL, request: requestURL}, nil)
		}
		if match {
			return &cfg, nil
		}
	}
	return nil, iodine.New(errNoMatchingHost{}, nil)
}

//getBashCompletionCmd generates bash completion file.
// TODO don't kill, return an error instead. caller should kill, not this function
func getBashCompletionCmd() {
	var b bytes.Buffer
	if os.Getenv("SHELL") != "/bin/bash" {
		console.Fatalln("Unsupported shell for bash completion detected.. exiting")
	}
	b.WriteString(mcBashCompletion)
	f, _ := getMcBashCompletionFilename()
	// TODO uncomment when ready
	//	if err != nil {
	//		return err
	//	}
	fl, err := os.OpenFile(f, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	defer fl.Close()
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalln("Unable to create bash completion file")
	}
	_, err = fl.Write(b.Bytes())
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalln("Unable to write bash completion file")
	}
	msg := "\nConfiguration written to " + f
	msg = msg + "\n\n$ source ${HOME}/.mc/mc.bash_completion\n"
	msg = msg + "$ echo 'source ${HOME}/.mc/mc.bash_completion' >> ${HOME}/.bashrc"
	console.Infoln(msg)
}

// saveConfigCmd writes config file to disk
func saveConfigCmd(ctx *cli.Context) {
	err := saveConfig(ctx)
	if os.IsExist(iodine.ToError(err)) {
		log.Debug.Println(iodine.New(err, nil))
		configPath, _ := getMcConfigPath()
		console.Fatalln("mc: Configuration file " + configPath + " already exists")
	}

	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		configPath, _ := getMcConfigPath()
		console.Fatalln("mc: Unable to generate config file", configPath)
	}
	configPath, err := getMcConfigPath()
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalln("mc: Unable to identify config file path")
	}
	console.Infoln("Configuration written to " + configPath + ". Please update your access credentials.")
}

// doConfigCmd is the handler for "mc config" sub-command.
func doConfigCmd(ctx *cli.Context) {
	if len(ctx.Args()) < 1 {
		cli.ShowCommandHelpAndExit(ctx, "config", 1) // last argument is exit code
	}
	switch true {
	case ctx.Bool("completion") == true:
		getBashCompletionCmd()
	default:
		saveConfigCmd(ctx)
	}
}
