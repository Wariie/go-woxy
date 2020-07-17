package core

import (
	"fmt"
	"log"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"reflect"
	"strings"

	com "github.com/Wariie/go-woxy/com"
	"github.com/gin-gonic/gin"
)

/*ModuleConfig - Module configuration */
type ModuleConfig struct {
	NAME    string
	VERSION int
	TYPES   string
	EXE     ModuleExecConfig
	BINDING ServerConfig
	STATE   string
	pk      string
}

//GetServer -
func (mc *ModuleConfig) GetServer() com.Server {
	return com.Server{IP: mc.BINDING.ADDRESS, Path: mc.BINDING.PATH[0], Port: mc.BINDING.PORT, Protocol: mc.BINDING.PROTOCOL}
}

//Stop -
func (mc *ModuleConfig) Stop() int {

	if mc.STATE != "ONLINE" {
		return -1
	}
	var sr com.ShutdownRequest
	sr.Hash = mc.pk
	sr.Name = mc.NAME
	r := com.SendRequest(mc.GetServer(), &sr, false)
	log.Println(r)
	//TODO BEST GESTURE
	if true {
		mc.STATE = "STOPPED"
	}
	return 0
}

//Setup - Setup module from config
func (mc *ModuleConfig) Setup(router *gin.Engine) error {
	fmt.Println("Setup mod : ", mc)
	if !reflect.DeepEqual(mc.EXE, ModuleExecConfig{}) {
		if strings.Contains(mc.EXE.SRC, "http") || strings.Contains(mc.EXE.SRC, "git@") {
			mc.Download()
		}
		go mc.Start()
	} else {
		log.Println("LOCAL BUILD or NO BUILD")
	}

	return mc.Hook(router)
}

//Start - Start module with config args and auto args
func (mc *ModuleConfig) Start() {
	mc.STATE = "LAUNCHING"
	//logFileName := mc.NAME + ".txt"

	fmt.Println("Starting mod : ", mc)
	cmd := exec.Command("go", "run", mc.EXE.MAIN)
	cmd.Dir = mc.EXE.BIN
	output, err := cmd.Output()
	if err == nil {
		log.Println(string(output), err)
	}
	log.Println(string(output), err)
}

//Download - Download module from repository ( git clone )
func (mc *ModuleConfig) Download() {

	//fmt.Println("Downloading mod : ", mc.NAME)
	if mc.STATE != "ONLINE" {
		wd, err := os.Getwd()

		var listArgs []string
		var action string

		if _, err := os.Stat(wd + "/mods" + "/" + mc.NAME); !os.IsExist(err) {
			//os.RemoveAll(wd + "/mods" + "/" + mc.NAME)
			listArgs = []string{"clone", mc.EXE.SRC}
			action = "Downloaded"
		} else {
			listArgs = []string{"pull"}
			action = "Update"
		}

		cmd := exec.Command("git", listArgs...)
		cmd.Dir = wd + "/mods"
		out, err := cmd.Output()
		fmt.Println(action, " mod : ", mc, " - ", string(out), " ", err)

		mc.EXE.BIN = wd + "/mods/" + mc.NAME + "/" + mc.EXE.BIN
		mc.STATE = "DOWNLOADED"
	} else {
		log.Fatalln("Error - Trying to download/update module while running\nStop it before")
	}
}

//Hook - Create a binding between module config address and gin server
func (mc *ModuleConfig) Hook(router *gin.Engine) error {
	paths := mc.BINDING.PATH

	if len(paths) > 0 && len(paths[0]) > 0 {
		for i := range paths {
			if len(paths[i]) > 0 {
				router.GET(paths[i], ReverseProxy(mc, paths[i]))
				fmt.Println("Module " + mc.NAME + " Hooked to Go-Proxy Server at - " + paths[i])
			}
		}
	}
	return nil
}

//ReverseProxy - reverse proxy for mod
func ReverseProxy(mc *ModuleConfig, path string) gin.HandlerFunc {
	return func(c *gin.Context) {
		mod := GetManager().config.MODULES[mc.NAME]
		if mod.STATE == "ONLINE" {
			if mod.BINDING.ROOT != "" {
				c.File(mod.BINDING.ROOT)
			} else if strings.Contains(mod.TYPES, "web") {
				url, err := url.Parse(mod.BINDING.PROTOCOL + "://" + mod.BINDING.ADDRESS + ":" + mod.BINDING.PORT)
				if err != nil {
					log.Println(err)
				}
				proxy := httputil.NewSingleHostReverseProxy(url)
				proxy.ServeHTTP(c.Writer, c.Request)
			}
		} else {
			c.String(503, "MODULE LOADING WAIT A SECOND PLEASE ....")
		}
	}
}

/*Config - Global configuration */
type Config struct {
	NAME    string
	MODULES map[string]ModuleConfig
	VERSION int
	SERVER  ServerConfig
}

/*ModuleExecConfig - Module exec file informations */
type ModuleExecConfig struct {
	SRC  string
	MAIN string
	BIN  string
}

/*ServerConfig - Server configuration*/
type ServerConfig struct {
	ADDRESS  string
	PORT     string
	PATH     []string
	PROTOCOL string
	ROOT     string
}
