package core

import (
	"bufio"
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

func readConfig(configPath string) Config {

	fmt.Println("Read config file at " + configPath)

	if configPath != "" {
		configFile = configPath
	}

	//READ CONFIG FILE
	data, errR := ioutil.ReadFile(configFile)
	if errR != nil {
		log.Fatalf("error: %v", errR)
	}

	t := Config{}

	//PARSE CONFIG FILE
	err := yaml.Unmarshal(data, &t)
	if err != nil || t.NAME == "" {
		log.Fatalf("error: %v", err)
	}

	t.SERVER = checkServerConfig(t.SERVER)

	t.MODULES = checkModulesConfig(t.MODULES)

	return t
}

func checkModulesConfig(mc map[string]ModuleConfig) map[string]ModuleConfig {

	for k := range mc {
		m := mc[k]
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
		mc[k] = m
	}
	return mc
}

func checkServerConfig(sc ServerConfig) ServerConfig {

	//CHECK IP IF NOT PRESENT -> DEFAULT LOCALHOST
	if sc.ADDRESS == "" {
		sc.ADDRESS = "0.0.0.0"
	}

	//CHECK PORT IF NOT PRESENT -> DEFAULT 2000
	if sc.PORT == "" {
		sc.PORT = "2000"
	}

	return sc
}

func loadModules() {
	//INIT MODULE DIRECTORY
	wd, err := os.Getwd()

	os.Mkdir(wd+"/mods", os.ModeDir)
	if err != nil {
		log.Fatalln("Error creating mods folder : ", err)
		os.Exit(1)
	}

	config := GetManager().config
	Router := GetManager().router
	for k := range config.MODULES {
		mod := config.MODULES[k]
		err := mod.Setup(Router, true)
		if err != nil {
			log.Fatalln("Error setup module ", mod.NAME, " - ", err)
		}
		config.MODULES[k] = mod
	}

	//ADD HUB MODULE FOR COMMAND GESTURE
	config.MODULES["hub"] = ModuleConfig{NAME: "hub", PK: "hub"}

	GetManager().router = Router
	GetManager().config = config
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

func generateSecret() {
	s := tools.String(64)
	sb := []byte(s)
	err := ioutil.WriteFile(".secret", sb, 0644)
	if err != nil {
		log.Println("Error trying create secret file :", err)
	}
	/*b := sha256.Sum256(s)
	secretHash = string(b[:])*/
	secretHash = s
}

func motd() {
	fmt.Println(" -------------------- Go-Woxy - V 0.0.1 -------------------- ")
	file, err := os.Open(motdFileName)
	if err != nil {
		log.Fatalln("No motd file ", motdFileName, " : ", err)
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
