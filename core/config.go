package core

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

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
		m.STATE = "UNKNOW"

		if strings.Contains(m.TYPES, "bind") {
			m.STATE = "ONLINE"
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
