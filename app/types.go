package app

import (
	"log"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/gin-gonic/gin"
	com "guilhem-mateo.fr/git/Wariie/go-woxy.git/app/com"
)

/*ModuleConfig - Module configuration */
type ModuleConfig struct {
	NAME      string
	VERSION   int
	SRC       string
	BIN       string
	MAIN_FILE string
	SERVER    ServerConfig
	STATE     string
	pk        string
}

//GetServer -
func (mc *ModuleConfig) GetServer() com.Server {
	return com.Server{IP: mc.SERVER.ADDRESS, Path: mc.SERVER.PATH, Port: mc.SERVER.PORT, Protocol: mc.SERVER.PROTOCOL}
}

//Stop -
func (mc *ModuleConfig) Stop() int {
	if mc.STATE != "ONLINE" {
		return -1
	}
	var sr com.ShutdownRequest
	sr.Hash = mc.pk
	sr.Name = mc.NAME
	r := com.SendRequest(mc.GetServer(), &sr)
	log.Println(r)
	//TODO BETTER HANDLING RESULT

	return 0
}

//Setup - Setup module from config
func (mc *ModuleConfig) Setup(router *gin.Engine) error {
	log.Println("Setup mod : ", mc)
	if strings.Contains(mc.SRC, "http") || strings.Contains(mc.SRC, "git@") {
		log.Println("Downloading mod : ", mc.NAME)
		mc.Download()
	} else {
		log.Println("LOCAL BUILD or NO BUILD")
	}

	return mc.Hook(router)
}

//Start - Start module with config args and auto args
func (mc *ModuleConfig) Start() {
	mc.STATE = "LAUNCHING"
	//logFileName := mc.NAME + ".txt"

	log.Println("Starting mod : ", mc)
	cmd := exec.Command("go", "run", mc.MAIN_FILE)
	cmd.Dir = mc.BIN
	output, err := cmd.Output()
	if err == nil {
		log.Println(string(output), err)
	}
	log.Println(string(output), err)
}

//Download - Download module from repository
func (mc *ModuleConfig) Download() {

	cmd := exec.Command("git", "clone", mc.SRC)
	wd, err := os.Getwd()

	//WORKING DIR + "/mods" (NEED ALREADY CREATED DIR (DO AT STARTUP ?))
	cmd.Dir = wd + "/mods"

	if _, err := os.Stat(cmd.Dir + "/" + mc.NAME); !os.IsNotExist(err) {
		os.RemoveAll(cmd.Dir + "/" + mc.NAME)
	}

	out, err := cmd.Output()
	log.Println("Downloaded mod : ", mc, " - ", string(out), " ", err)

	mc.BIN = wd + "/mods/" + mc.NAME + "/" + mc.BIN
	mc.STATE = "DOWNLOADED"
}

//Hook -
func (mc *ModuleConfig) Hook(router *gin.Engine) error {
	paths := strings.Split(mc.SERVER.PATH, ";")

	/*for mc.STATE != "DOWNLOADED" {
		time.Sleep(time.Second * 2)
	}*/

	if len(paths) > 1 && len(paths[0]) > 0 {
		for i := range paths {
			if len(paths[i]) > 0 {
				router.GET(paths[i], ReverseProxy(mc, paths[i]))
				log.Print("Module " + mc.NAME + " Hooked to Go-Proxy Server at - " + paths[i])
			}
		}
	}
	return nil
}

//ReverseProxy - reverse proxy for mod
func ReverseProxy(mc *ModuleConfig, path string) gin.HandlerFunc {
	return func(c *gin.Context) {
		url, err := url.Parse(mc.SERVER.PROTOCOL + "://" + mc.SERVER.ADDRESS + ":" + mc.SERVER.PORT)
		if err != nil {
			log.Println(err)
		}
		proxy := httputil.NewSingleHostReverseProxy(url)
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

/*Config - Global configuration */
type Config struct {
	NAME    string
	MODULES map[string]ModuleConfig
	VERSION int
	SERVER  ServerConfig
}

/*ServerConfig - Server configuration*/
type ServerConfig struct {
	ADDRESS  string
	PORT     string
	PATH     string
	PROTOCOL string
}
