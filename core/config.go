package core

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"log"
	"os"
	"strings"

	"github.com/Wariie/go-woxy/tools"
	"github.com/spf13/viper"
)

/*Config - Global configuration */
type Config struct {
	ACCESSLOGFILE string
	MODULES       map[string]ModuleConfig
	MOTD          string
	NAME          string
	SECRET        string
	MODDIR        string
	RESOURCEDIR   string
	SERVER        ServerConfig
	VERSION       int
}

func (c *Config) LoadConfig(path string) (err error) {
	//EMPTY CONFIG FILE PATH
	if len(path) == 0 {
		//TRY DEFAULT cfg.yml
		path = "cfg.yml"
	}

	viper.AddConfigPath(path)
	viper.AddConfigPath(".")
	viper.SetConfigName("cfg")
	viper.SetConfigType("yml")

	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&c)

	c.checkServer()

	c.checkModules()

	if c.RESOURCEDIR == "" {
		c.RESOURCEDIR = "resources" + string(os.PathSeparator)
	}
	if c.MODDIR == "" {
		c.MODDIR = "mods" + string(os.PathSeparator)
	}

	// Convert map to slice of values.
	log.Println("GO-WOXY Core - Config loaded")

	return err
}

func (c *Config) checkModules() {
	for k, m := range c.MODULES {
		m.NAME = k

		if strings.Contains(m.TYPES, "bind") {
			m.STATE = Online
		} else {
			m.STATE = Unknown
		}

		if m.LOG.Enabled == nil {
			enabled := true
			m.LOG.Enabled = &enabled
			m.LOG.Path = "default"
		}

		if m.BINDING.PROTOCOL == "" {
			m.BINDING.PROTOCOL = "http"
		}

		if m.BINDING.ADDRESS == "" {
			m.BINDING.ADDRESS = "0.0.0.0"
		}

		c.MODULES[k] = m
	}
}

func (c *Config) checkServer() {

	//CHECK IP IF NOT PRESENT -> DEFAULT LOCALHOST
	if c.SERVER.ADDRESS == "" {
		c.SERVER.ADDRESS = "0.0.0.0"
	}

	//CHECK PORT IF NOT PRESENT -> DEFAULT 2000
	if c.SERVER.PORT == "" {
		c.SERVER.PORT = "2000"
	}
}

func (c *Config) generateSecret() {
	if len(c.SECRET) == 0 {
		b := []byte(tools.String(64))
		err := os.WriteFile(".secret", b, 0644)
		if err != nil {
			log.Fatalln("GO-WOXY Core - Error creating secret file : ", err)
		}
		h := sha256.New()
		h.Write(b)
		c.SECRET = base64.URLEncoding.EncodeToString(h.Sum(nil))
	}
}

// GetMotdFileContent - Get motd file content from motd path
func (c *Config) GetMotdFileContent() string {
	if c.MOTD == "" {
		c.MOTD = "motd.txt"
	}

	file, err := os.Open(c.MOTD)
	if err != nil {
		log.Panicln("GO-WOXY Core - Error cannot found ", c.MOTD, " : ", err)
	}
	defer file.Close()
	motdContent := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		motdContent += scanner.Text() + "\n"
	}

	return motdContent
}
