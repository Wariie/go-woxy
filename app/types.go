package app

import (
	"log"
	"net/http/httputil"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"guilhem-mateo.fr/go-woxy/app/com"
)

/*ModuleConfig - Module configuration */
type ModuleConfig struct {
	NAME    string
	VERSION int
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

//BuildAndStart -
func (mc *ModuleConfig) BuildAndStart() int {

	mc.STATE = "BUILDING"

	cmd := exec.Command("go", "build")
	cmd.Dir = mc.BIN
	out, err := cmd.Output()
	log.Println("	Loading mod : ", mc, " - ", string(out), " ", err)

	mc.BIN = mc.BIN + mc.NAME + ".exe"
	//TODO BETTER RESULT HANDLING
	if err == nil {
		mc.STATE = "LAUNCHING"
		cmd := exec.Command(mc.BIN)
		err = cmd.Start()
		log.Println("	Launching mod : ", mc, " - ", err)
		if err == nil {
			return 0
		}
	}
	//mc.STATE = "ERROR"
	log.Println("	Error building ", mc.NAME, " : ", string(out), err)
	return -1
}

//Hook -
func (mc *ModuleConfig) Hook(router *gin.Engine) int {
	paths := strings.Split(mc.SERVER.PATH, ";")

	for mc.STATE != "LAUNCHING" {
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
	return 0
}

//ReverseProxy - reverse proxy for mod
func ReverseProxy(mc *ModuleConfig, path string) gin.HandlerFunc {
	return func(c *gin.Context) {
		/*director := func(req *http.Request) {

			log.Println("URL : ", url)
			if err != nil {
				log.Println("ERR PARSING URL ", err)
			}
			req.URL.Scheme = url.Scheme
			req.Host, req.URL.Host = url.Host, url.Host
			req.URL.Path = path
			req.Header["my-header"] = []string{req.Header.Get("my-header")}
			// Golang camelcases headers
			delete(req.Header, "My-Header")
		}*/
		url, err := url.Parse(mc.SERVER.PROTOCOL + "://" + mc.SERVER.ADDRESS + ":" + mc.SERVER.PORT)
		if err != nil {
			log.Println(err)
		}
		proxy := httputil.NewSingleHostReverseProxy(url) //httputil.ReverseProxy{Director: director}
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
