package core

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"github.com/Wariie/go-woxy/com"
	auth "github.com/abbot/go-http-auth"
	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/process"
)

//Download - Download module from repository ( git clone )
func (mc *ModuleConfig) Download() {

	if mc.STATE != Online {
		var listArgs []string
		var action string

		wd := "./mods/"
		if _, err := os.Stat(wd + mc.NAME + "/"); os.IsNotExist(err) {
			listArgs = []string{"clone", mc.EXE.SRC}
			action = "Downloaded"
		} else {
			listArgs = []string{"pull"}
			action = "Update"
			wd += mc.NAME + "/"
		}

		cmd := exec.Command("git", listArgs...)
		cmd.Dir = wd
		out, err := cmd.CombinedOutput()
		fmt.Println(action, " mod : ", mc, " - ", string(out), " ", err)

		mc.EXE.BIN = "./mods/" + mc.NAME + "/"
		mc.STATE = Downloaded
	} else {
		log.Println("Error - Trying to download/update module while running\nStop it before")
	}
}

//GetLog - GetLog from Module
func (mc *ModuleConfig) GetLog() string {

	file, err := os.Open("./mods/" + mc.NAME + "/log.log")
	if err != nil {
		log.Fatalln("failed reading file :", err)
	}
	b, err := ioutil.ReadAll(file)
	return string(b)
}

//GetPerf - GetPerf from Module
func (mc *ModuleConfig) GetPerf() (float64, float32) {
	p, err := process.NewProcess(int32(mc.pid))
	ram, err := p.MemoryPercent()
	cpu, err := p.Percent(0)
	name, err := p.Name()
	log.Println("PERF :", name, err)

	return cpu, ram
}

//GetServer - Get Module Server configuration
func (mc *ModuleConfig) GetServer(path string) com.Server {
	if path == "" {
		path = mc.BINDING.PATH[0].FROM
	}
	return com.Server{IP: com.IP(mc.BINDING.ADDRESS), Path: com.Path(path), Port: com.Port(mc.BINDING.PORT), Protocol: com.Protocol(mc.BINDING.PROTOCOL)}
}

//HookAll - Create all binding between module config address and gin server
func (mc *ModuleConfig) HookAll(router *gin.Engine) error {
	paths := mc.BINDING.PATH

	if strings.Contains(mc.TYPES, "web") {
		sP := ""
		if len(paths[0].FROM) > 1 {
			sP = paths[0].FROM
		}
		r := Route{FROM: sP + "/ressources/*filepath", TO: "/ressources/*filepath"}
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
		if mc.AUTH.ENABLED {
			htpasswd := auth.HtpasswdFileProvider(".htpasswd")
			authenticator := auth.NewBasicAuthenticator("Some Realm", htpasswd)
			authorized := router.Group("/", BasicAuth(authenticator))
			authorized.Handle("GET", r.FROM, ReverseProxy(mc.NAME, r))
		} else {
			router.Handle("GET", r.FROM, ReverseProxy(mc.NAME, r))
		}
		fmt.Println("GO-WOXY Core - Module " + mc.NAME + " Hooked to Go-Proxy Server at - " + r.FROM + " => " + r.TO)
	}
	return nil
}

//Setup - Setup module from config
func (mc *ModuleConfig) Setup(router *gin.Engine, hook bool) error {
	fmt.Println("GO-WOXY Core - Setup mod : ", mc)
	if !mc.EXE.REMOTE && !reflect.DeepEqual(mc.EXE, ModuleExecConfig{}) {
		if strings.Contains(mc.EXE.SRC, "http") || strings.Contains(mc.EXE.SRC, "git@") {
			mc.Download()
		}
		mc.copySecret()
		go mc.Start()
	} // ELSE NO BUILD

	if hook {
		return mc.HookAll(router)
	}
	return nil
}

//Start - Start module with config args and auto args
func (mc *ModuleConfig) Start() {
	mc.STATE = Loading

	var platformParam []string
	if runtime.GOOS == "windows" {
		platformParam = []string{"cmd", "/c ", "go", "run", mc.EXE.MAIN, "1>", "log.log", "2>&1"}
	} else {
		platformParam = []string{"/bin/sh", "-c", "go run " + mc.EXE.MAIN + " > log.log 2>&1"}
	}

	fmt.Println("GO-WOXY Core - Starting mod : ", mc)
	cmd := exec.Command(platformParam[0], platformParam[1:]...)
	cmd.Dir = mc.EXE.BIN
	output, err := cmd.Output()
	if err != nil {
		log.Println("GO-WOXY Core - Error:", err)
	}
	log.Println("GO-WOXY Core - Output :", string(output), err)
}

func (mc *ModuleConfig) copySecret() {
	source, err := os.Open(".secret")
	if err != nil {
		log.Println("GO-WOXY Core - Error reading generated secret file")
	}
	defer source.Close()

	destination, err := os.Create(mc.EXE.BIN + "/.secret")
	if err != nil {
		log.Println("GO-WOXY Core - Error creating mod secret file")
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	if err != nil {
		log.Println("GO-WOXY Core - Error Copy Secret:", err, nBytes)
	}
}

// BasicAuth - Authentification gin middleware
func BasicAuth(a *auth.BasicAuth) gin.HandlerFunc {
	realmHeader := "Basic realm=" + strconv.Quote(a.Realm)

	return func(c *gin.Context) {
		user := a.CheckAuth(c.Request)
		if user == "" {
			c.Header("WWW-Authenticate", realmHeader)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Set("user", user)
	}
}

// NewSingleHostReverseProxy -
func NewSingleHostReverseProxy(target *url.URL) *httputil.ReverseProxy {
	targetQuery := target.RawQuery
	director := func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
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

//ReverseProxy - reverse proxy for mod
func ReverseProxy(modName string, r Route) gin.HandlerFunc {
	return func(c *gin.Context) {
		mod := GetManager().config.MODULES[modName]

		//CHECK IF MODULE IS ONLINE
		if mod.STATE == Online {
			//IF ROOT IS PRESENT REDIRECT TO IT
			if strings.Contains(mod.TYPES, "bind") && mod.BINDING.ROOT != "" {
				c.File(mod.BINDING.ROOT)

			} else if strings.Contains(mod.TYPES, "web") {
				//ELSE IF BINDING IS TYPE **WEB**
				//REVERSE PROXY TO IT
				url, err := url.Parse(mod.BINDING.PROTOCOL + "://" + mod.BINDING.ADDRESS + ":" + mod.BINDING.PORT + r.TO)
				if err != nil {
					log.Println(err)
				}
				proxy := NewSingleHostReverseProxy(url)
				proxy.ServeHTTP(c.Writer, c.Request)
			}
			//TODO HANDLE MORE STATES
		} else {
			title := ""
			code := 500
			message := ""
			if mod.STATE == Loading || mod.STATE == Downloaded {
				title = "Loading"
				code += 3
				message = "Module is loading ..."
			} else if mod.STATE == Stopped {
				title = "Stopped"
				code = 410
				message = "Module stopped by an administrator"
			} else if mod.STATE == Error || mod.STATE == Unknown {
				title = "Error"
				message = "Error"
			}
			c.HTML(code, "loading.html", gin.H{
				"title":   title,
				"code":    code,
				"message": message,
			})
		}
	}
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
	MODULES map[string]ModuleConfig
	MOTD    string
	NAME    string
	SECRET  string
	SERVER  ServerConfig
	VERSION int
}

/*ModuleConfig - Module configuration */
type ModuleConfig struct {
	AUTH     ModuleAuthConfig
	BINDING  ServerConfig
	COMMANDS []string
	EXE      ModuleExecConfig
	NAME     string
	pid      int
	PK       string
	STATE    ModuleState
	TYPES    string
	VERSION  int
}

/*ModuleExecConfig - Module exec file informations */
type ModuleExecConfig struct {
	BIN        string
	MAIN       string
	REMOTE     bool
	SRC        string
	SUPERVISED bool
}

/*ServerConfig - Server configuration*/
type ServerConfig struct {
	ADDRESS  string
	PATH     []Route
	PORT     string
	PROTOCOL string
	ROOT     string
	CERT     string
	CERT_KEY string
}

/*ModuleAuthConfig - Auth configuration*/
type ModuleAuthConfig struct {
	ENABLED bool
	TYPE    string
}

// Route - Route redirection
type Route struct {
	FROM string
	TO   string
}

//ModuleState - State of ModuleConfig
type ModuleState string

const (
	Unknown    ModuleState = "UNKNOWN"
	Loading    ModuleState = "LOADING"
	Online     ModuleState = "ONLINE"
	Stopped    ModuleState = "STOPPED"
	Downloaded ModuleState = "DOWNLOADED"
	Error      ModuleState = "ERROR"
	Failed     ModuleState = "FAILED"
)
