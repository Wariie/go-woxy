package core

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
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

	fmt.Println("Read config file at " + configPath)

	//READ CONFIG FILE
	data, errR := ioutil.ReadFile(configPath)
	if errR != nil {
		log.Fatalf("Error config file : %v", errR)
	}

	//PARSE CONFIG FILE
	err := yaml.Unmarshal(data, &c)
	if err != nil || c.NAME == "" {
		log.Fatalf("error: %v", err)
	}

	c.checkServer()

	c.checkModules()
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
		log.Fatalln("Error creating mods folder : ", err)
		os.Exit(1)
	}

	Router := GetManager().router
	for k := range c.MODULES {
		mod := c.MODULES[k]
		err := mod.Setup(Router, true)
		if err != nil {
			log.Fatalln("Error setup module ", mod.NAME, " - ", err)
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

func getServerConfig(sc ServerConfig, router *gin.Engine) http.Server {
	path := ""
	if len(sc.PATH) > 0 {
		path = sc.PATH[0].FROM
	}
	fmt.Println("SERVER ADDRESS : \"" + sc.ADDRESS + ":" + sc.PORT + path + "\"")
	return http.Server{
		Addr:    sc.ADDRESS + ":" + sc.PORT + path,
		Handler: router,
	}
}

func (c *Config) generateSecret() {
	if c.SECRET == "" {
		s := tools.String(64)
		err := ioutil.WriteFile(".secret", []byte(s), 0644)
		if err != nil {
			log.Println("Error trying create secret file :", err)
		}
		c.SECRET = s
	}
}

func generatePrivateKey() {

	//GENERATE PRIVATE KEY
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	//SAVE PRIVATE KEY AS PEM FILE
	pemPrivateKeyFile, err := os.Create("private.key")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var pemkey = &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}

	err = pem.Encode(pemPrivateKeyFile, pemkey)

	//SAVE PUBLIC KEY AS PEM FILE
	pemPublicKeyFile, err := os.Create("public.pem")

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// http://golang.org/pkg/encoding/pem/#Block
	var pemPublicKey = &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(&privateKey.PublicKey)}

	err = pem.Encode(pemPublicKeyFile, pemPublicKey)
}

func (c *Config) motd() {
	if c.MOTD == "" {
		c.MOTD = "motd.txt"
	}

	fmt.Println(" -------------------- Go-Woxy - V 0.0.1 -------------------- ")
	file, err := os.Open(c.MOTD)
	if err != nil {
		log.Fatalln("Error - Cannot found ", c.MOTD, " : ", err)
		return
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
