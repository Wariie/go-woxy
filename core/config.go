package core

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/Wariie/go-woxy/tools"
	"gopkg.in/yaml.v2"
)

/*Config - Global configuration */
type Config struct {
	ACCESSLOGFILE string
	MODULES       map[string]ModuleConfig
	modulesList   []*ModuleConfig
	MOTD          string
	NAME          string
	SECRET        string
	MODDIR        string
	RESOURCEDIR   string
	SERVER        ServerConfig
	VERSION       int
}

func (c *Config) Load(configPath string) {

	if len(configPath) == 0 {
		//EMPTY CONFIG FILE PATH
		//TRY DEFAULT cfg.yml
		configPath = "cfg.yml"
	}

	//READ CONFIG FILE
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Fatalf("GO-WOXY Core - Error reading config file : %v", err)
	}

	//PARSE CONFIG FILE
	err = yaml.Unmarshal(data, &c)
	if err != nil || c.NAME == "" {
		log.Fatalf("GO-WOXY Core - Error parsing config file %v", err)
	}

	c.checkServer()

	c.checkModules()

	if c.RESOURCEDIR == "" {
		c.RESOURCEDIR = "resources" + string(os.PathSeparator)
	}
	if c.MODDIR == "" {
		c.MODDIR = "mods" + string(os.PathSeparator)
	}

	// Convert map to slice of values.

	for _, mod := range c.MODULES {
		c.modulesList = append(c.modulesList, &mod)
	}

	log.Println("GO-WOXY Core - Config file readed")
}

func (c *Config) checkModules() {
	for k, m := range c.MODULES {
		m.NAME = k

		if strings.Contains(m.TYPES, "bind") {
			m.STATE = Online
		} else {
			m.STATE = Unknown
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
	if c.SECRET == "" {
		b := []byte(tools.String(64))
		err := ioutil.WriteFile(".secret", b, 0644)
		if err != nil {
			log.Fatalln("GO-WOXY Core - Error creating secret file : ", err)
		}
		h := sha256.New()
		h.Write(b)
		c.SECRET = base64.URLEncoding.EncodeToString(h.Sum(nil))
	}
}

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
