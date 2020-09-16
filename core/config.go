package core

import (
	"bufio"
	"crypto/sha256"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/Wariie/go-woxy/tools"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v2"
)

func (c *Config) loadConfig(configPath string) {

	if configPath == "" {
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

	fmt.Println("GO-WOXY Core - Config file readed")
}

func (c *Config) checkModules() {
	for k := range c.MODULES {
		m := c.MODULES[k]
		m.NAME = k
		m.STATE = Unknown

		if strings.Contains(m.TYPES, "bind") {
			m.STATE = Online
		}

		if m.BINDING.PROTOCOL == "" {
			m.BINDING.PROTOCOL = "http"
		}

		if m.BINDING.ADDRESS == "" {
			m.BINDING.ADDRESS = "127.0.0.1"
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

func (c *Config) loadModules() {
	//INIT MODULE DIRECTORY
	wd, err := os.Getwd()

	os.Mkdir(wd+"/mods", os.ModeDir)
	if err != nil {
		log.Fatalln("GO-WOXY Core - Error creating mods folder : ", err)
		os.Exit(1)
	}

	Router := GetManager().router
	for k := range c.MODULES {
		mod := c.MODULES[k]
		err := mod.Setup(Router, true)
		if err != nil {
			log.Fatalln("GO-WOXY Core - Error setup module ", mod.NAME, " : ", err)
		}
		c.MODULES[k] = mod
	}

	//ADD HUB MODULE FOR COMMAND GESTURE
	c.MODULES["hub"] = ModuleConfig{NAME: "hub", PK: "hub"}

	GetManager().router = Router
	GetManager().config = c
}

func initSupervisor() {
	s := Supervisor{}
	GetManager().SetSupervisor(&s)
	go s.Supervise()
}

func (c *Config) configAndServe(router *gin.Engine) error {
	path := ""
	if len(c.SERVER.PATH) > 0 {
		path = c.SERVER.PATH[0].FROM
	}
	fmt.Println("GO-WOXY Core - Serving at " + c.SERVER.PROTOCOL + "://" + c.SERVER.ADDRESS + ":" + c.SERVER.PORT + path)

	var s http.Server
	//CHECK FOR CERTIFICATE TO TRY TLS CONFIG
	if c.SERVER.CERT != "" && c.SERVER.CERT_KEY != "" {
		tls, err := c.getTLSConfig()
		if err != nil {
			log.Fatalln("GO-WOXY Core - Error tls config")
		}
		s = http.Server{
			Addr:      c.SERVER.ADDRESS + ":" + c.SERVER.PORT + path,
			Handler:   router,
			TLSConfig: tls,
		}
	} else {
		s = http.Server{
			Addr:    c.SERVER.ADDRESS + ":" + c.SERVER.PORT + path,
			Handler: router,
		}
	}
	return s.ListenAndServe()
}

func (c *Config) generateSecret() {
	if c.SECRET == "" {
		b := []byte(tools.String(64))
		err := ioutil.WriteFile(".secret", b, 0644)
		if err != nil {
			log.Fatalln("GO-WOXY Core - Error trying create secret file :", err)
		}
		h := sha256.New()
		h.Write(b)
		c.SECRET = string(h.Sum(nil))
	}
}

func (c *Config) getTLSConfig() (*tls.Config, error) {

	cer, err := tls.LoadX509KeyPair(c.SERVER.CERT, c.SERVER.CERT_KEY)
	if err != nil {
		log.Println(err)
		return &tls.Config{}, err
	}

	return &tls.Config{Certificates: []tls.Certificate{cer}}, nil
}

func (c *Config) motd() {
	if c.MOTD == "" {
		c.MOTD = "motd.txt"
	}

	fmt.Println(" -------------------- Go-Woxy - V 0.0.1 -------------------- ")
	file, err := os.Open(c.MOTD)
	if err != nil {
		log.Panicln("GO-WOXY Core - Error cannot found ", c.MOTD, ":", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}
	fmt.Println("------------------------------------------------------------ ")
}

func searchModWithHash(hash string) ModuleConfig {
	mods := GetManager().config.MODULES
	for i := range mods {
		if mods[i].PK == hash {
			return mods[i]
		}
	}
	return ModuleConfig{NAME: "error"}
}
