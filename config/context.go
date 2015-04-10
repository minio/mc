package config

import (
	"encoding/json"
	"io"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/minio-io/minio/pkg/iodine"
)

// Context - config context
type Context struct {
	// copy of Config
	Config *Config

	// file and path for config
	configFile string
	configPath string

	// lock
	lock *sync.RWMutex
}

// GetConfigPath path to config file
func (c *Context) GetConfigPath() string {
	return c.configPath
}

// GetConfigFile config file
func (c *Context) GetConfigFile() string {
	return c.configFile
}

// SaveConfig - write encoded json in config file
func (c *Context) SaveConfig() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	var file *os.File
	var err error

	file, err = os.OpenFile(c.configFile, os.O_WRONLY, 0666)
	defer file.Close()
	if err != nil {
		return iodine.New(err, nil)
	}

	encoder := json.NewEncoder(file)
	encoder.Encode(c.Config)
	return nil
}

// LoadConfig - read json config file and decode
func (c *Context) LoadConfig() error {
	c.lock.RLock()
	defer c.lock.RUnlock()

	var file *os.File
	var err error

	file, err = os.OpenFile(c.configFile, os.O_RDONLY, 0666)
	defer file.Close()
	if err != nil {
		return iodine.New(err, nil)
	}

	config := new(Config)
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	switch err {
	case io.EOF:
		return nil
	case nil:
		c.Config = config
		return nil
	default:
		return iodine.New(err, nil)
	}
}

func New(configPath, configFileName string) (*Context, error) {
	if configPath == "" || strings.TrimSpace(configPath) == "" {
		return nil, iodine.New(InvalidArgument{}, nil)
	}
	if configFileName == "" || strings.TrimSpace(configFileName) == "" {
		return nil, iodine.New(InvalidArgument{}, nil)
	}
	c := new(Context)
	c.configPath = configPath
	c.configFile = path.Join(c.configPath, configFileName)
	if _, err := os.Stat(c.configFile); os.IsNotExist(err) {
		_, err = os.Create(c.configFile)
		if err != nil {
			return nil, iodine.New(err, nil)
		}
	}
	c.lock = new(sync.RWMutex)
	return c, nil
}
