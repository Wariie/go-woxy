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
	"runtime"
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
func (mc *ModuleConfig) GetServer(path string) com.Server {
	if path == "" {
		path = mc.BINDING.PATH[0].FROM
	}
	return com.Server{IP: mc.BINDING.ADDRESS, Path: path, Port: mc.BINDING.PORT, Protocol: mc.BINDING.PROTOCOL}
}

//Stop -
func (mc *ModuleConfig) Stop() int {
	if mc.STATE != "ONLINE" {
		return -1
	}
	var sr com.ShutdownRequest
	sr.Hash = mc.pk
	sr.Name = mc.NAME
	r := com.SendRequest(mc.GetServer(""), &sr, false)
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
	var platformParam []string
	if runtime.GOOS == "windows" {
		platformParam = []string{"cmd", "/c"}
	} else {
		platformParam = []string{"/bin/sh", "-c"}
	}

	fmt.Println("Starting mod : ", mc)
	cmd := exec.Command(platformParam[0], platformParam[1], "go run "+mc.EXE.MAIN+" > log.log")
	cmd.Dir = mc.EXE.BIN
	output, err := cmd.Output()
	if err != nil {
		log.Println("Error:", err)
	}
	log.Println("Output :", string(output), err)
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
		r := Route{FROM: "/ressources/*filepath"}
		mc.Hook(router, r, "GET")
	}

	if len(paths) > 0 && len(paths[0].FROM) > 0 {
		for i := range paths {
			err := mc.Hook(router, paths[i], "GET")
			if err != nil {
				return err
			}
		}
	}

	return nil
}

//Hook - Create a binding between module and gin server
func (mc *ModuleConfig) Hook(router *gin.Engine, r Route, typeR string) error {
	if typeR == "" {
		typeR = "GET"
	}
	if len(r.FROM) > 0 {
		router.Handle("GET", r.FROM, ReverseProxy(mc, r))
		fmt.Println("Module " + mc.NAME + " Hooked to Go-Proxy Server at - " + r.FROM + " => " + r.TO)
	}
	return nil
}

//ReverseProxy - reverse proxy for mod
func ReverseProxy(mc *ModuleConfig, r Route) gin.HandlerFunc {
	return func(c *gin.Context) {
		mod := GetManager().config.MODULES[mc.NAME]

		//CHECK IF MODULE IS ONLINE
		if mod.STATE == "ONLINE" {
			//IF ROOT IS PRESENT REDIRECT TO IT
			if mod.BINDING.ROOT != "" {
				c.File(mod.BINDING.ROOT)

			} else if strings.Contains(mod.TYPES, "web") {
				//ELSE IF BINDING IS TYPE **WEB**
				//REVERSE PROXY TO IT
				query := ""
				if r.TO == "" {
					query = c.Request.URL.Path
				} else {
					query = r.TO
				}
				url, err := url.Parse(mod.BINDING.PROTOCOL + "://" + mod.BINDING.ADDRESS + ":" + mod.BINDING.PORT + query)
				if err != nil {
					log.Println(err)
				}
				proxy := NewSingleHostReverseProxy(url)
				proxy.ServeHTTP(c.Writer, c.Request)
			}
			//TODO HANDLE MORE STATES
		} else {
			//RETURN 503 WHILE MODULE IS LOADING
			c.String(503, "MODULE LOADING WAIT A SECOND PLEASE ....")
		}
		//GetManager().config.MODULES[mc.NAME] = mod
	}
}

// NewSingleHostReverseProxy -
func NewSingleHostReverseProxy(target *url.URL) *httputil.ReverseProxy {
	targetQuery := target.RawQuery
	director := func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = target.Path //singleJoiningSlash(target.Path, req.URL.Path)
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
	PATH     []Route
	PROTOCOL string
	ROOT     string
}

// Route - Route redirection
type Route struct {
	FROM string
	TO   string
}
