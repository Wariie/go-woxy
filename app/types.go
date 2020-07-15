package app

import (
	"log"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	com "guilhem-mateo.fr/git/Wariie/go-woxy.git/app/com"
)

/*ModuleConfig - Module configuration */
type ModuleConfig struct {
	NAME    string
	VERSION int
	SRC     string
	BIN     string
	SERVER  ServerConfig
	STATE   string
	pk      string
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

//Build - Build Module from src
func (mc *ModuleConfig) Build() error {
	mc.STATE = "BUILDING"

	cmd := exec.Command("go", "build")
	cmd.Dir = mc.BIN
	out, err := cmd.Output()
	log.Println("	Building mod : ", mc, " - ", string(out), " ", err)

	mc.BIN = mc.NAME

	if runtime.GOOS == "windows" {
		mc.BIN += ".exe"
	}
	return err
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

	if mc.BIN != "" {
		err := mc.Build()
		if err != nil {
			return err
		}
	}
	return mc.Hook(router)
}

//Start - Start module with config args and auto args
func (mc *ModuleConfig) Start() {
	mc.STATE = "LAUNCHING"
	//logFileName := mc.NAME + ".txt"

	cmd := exec.Command("pwd")
	b, err := cmd.Output()
	log.Println(string(b))

	startCmd := ""
	binPath := "./mods/" + mc.NAME + "/" + mc.BIN
	if runtime.GOOS == "windows" {
		startCmd = binPath //+ " > " + logFileName
	} else {
		startCmd = "nohup " + binPath + " &" // + " > " + logFileName + "2>&1"
	}

	cmd := exec.Command(startCmd)
	cmd.Stdout = os.Stdout
	err := cmd.Start()
	log.Println("Starting mod : ", mc, " - ", err)
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

	mc.BIN = wd + "/mods/" + mc.NAME
}

//Hook -
func (mc *ModuleConfig) Hook(router *gin.Engine) error {
	paths := strings.Split(mc.SERVER.PATH, ";")

	for mc.STATE != "BUILDING" {
		time.Sleep(time.Second * 2)
	}

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
