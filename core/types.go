package core

import (
	"fmt"
	"log"
	"net/http"
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

	return mc.HookAll(router)
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
		out, err := cmd.CombinedOutput()
		fmt.Println(action, " mod : ", mc, " - ", string(out), " ", err)

		mc.EXE.BIN = wd + "/mods/" + mc.NAME + "/" + mc.EXE.BIN
		mc.STATE = "DOWNLOADED"
	} else {
		log.Fatalln("Error - Trying to download/update module while running\nStop it before")
	}
}

//HookAll - Create all binding between module config address and gin server
func (mc *ModuleConfig) HookAll(router *gin.Engine) error {
	paths := mc.BINDING.PATH

	if strings.Contains(mc.TYPES, "web") {
		router.Static("/ressources", "./ressources")
	}

	if len(paths) > 0 && len(paths[0]) > 0 {
		for i := range paths {
			err := mc.Hook(router, paths[i], "", "GET")
			if err != nil {
				return err
			}
		}
	}

	return nil
}

//Hook - Create a binding between module and gin server
func (mc *ModuleConfig) Hook(router *gin.Engine, from string, to string, typeR string) error {
	if typeR == "" {
		typeR = "GET"
	}

	if to == "" {
		to = from
	}

	if len(from) > 0 {
		router.Handle("GET", from, ReverseProxy(mc, from, to))
		fmt.Println("Module " + mc.NAME + " Hooked to Go-Proxy Server at - " + from + " => " + to)
	}
	return nil
}

//ReverseProxy - reverse proxy for mod
func ReverseProxy(mc *ModuleConfig, path string, to string) gin.HandlerFunc {
	return func(c *gin.Context) {
		mod := GetManager().config.MODULES[mc.NAME]
		if mod.STATE == "ONLINE" {
			if mod.BINDING.ROOT != "" {
				c.File(mod.BINDING.ROOT)
			} else if strings.Contains(mod.TYPES, "web") {
				url, err := url.Parse(mod.BINDING.PROTOCOL + "://" + mod.BINDING.ADDRESS + ":" + mod.BINDING.PORT + path)
				if err != nil {
					log.Println(err)
				}

				//TODO REFACTOR REVERSO PROXY WITH CUSTOM ONE TO REWRITE HOST
				proxy := NewSingleHostReverseProxy(url)
				proxy.ServeHTTP(c.Writer, c.Request)
			}
		} else {
			c.String(503, "MODULE LOADING WAIT A SECOND PLEASE ....")
		}
	}
}

// NewSingleHostReverseProxy -
func NewSingleHostReverseProxy(target *url.URL) *httputil.ReverseProxy {
	targetQuery := target.RawQuery
	director := func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = singleJoiningSlash(target.Path, req.URL.Path)
		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}
		if _, ok := req.Header["User-Agent"]; !ok {
			// explicitly disable User-Agent so it's not set to default value
			req.Header.Set("User-Agent", "")
		}
	}
	return &httputil.ReverseProxy{Director: director}

}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b

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
