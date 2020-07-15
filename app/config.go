package app

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

func readConfig(configPath string) Config {

	log.Print("READ CONFIG")
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

	//defaultPort := 2420

	//SET MODULES NAME
	for k := range mc {
		m := mc[k]
		m.NAME = k
		m.STATE = "UNKNOW"

		if m.SERVER.PROTOCOL == "" {
			m.SERVER.PROTOCOL = "http"
		}

		if m.SERVER.ADDRESS == "" {
			m.SERVER.ADDRESS = "127.0.0.1"
		}

		/*if m.SERVER.PORT == "" {
			m.SERVER.PORT = strconv.FormatInt(int64(defaultPort), 10)
			defaultPort++
		}*/
		mc[k] = m
	}
	return mc
}

func checkServerConfig(sc ServerConfig) ServerConfig {

	//CHECK IP IF NOT PRESENT -> DEFAULT LOCALHOST
	if sc.ADDRESS == "" {
		sc.ADDRESS = "127.0.0.1"
	}

	//CHECK PORT IF NOT PRESENT -> DEFAULT 2000
	if sc.PORT == "" {
		sc.PORT = "2000"
	}

	return sc
}
