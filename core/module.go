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

func (mc *ModuleConfig) checkModuleRunning() bool {
	try := 0
	b := false
	for b == false && try < 5 {
		if mc.pid != 0 && (mc.EXE != ModuleExecConfig{}) && !mc.EXE.REMOTE {
			b = checkPidRunning(mc)
		}

		if !b {
			b = checkModulePing(mc)
		}
		try++
	}
	return b
}

//Download - Download module from repository ( git clone )
func (mc *ModuleConfig) Download(moduleDir string) {

	if mc.STATE != Online {
		pathSeparator := string(os.PathSeparator)
		fmt.Println("GO-WOXY Core - Downloading " + mc.NAME)
		var listArgs []string
		var action string
		path := ""
		if _, err := os.Stat(moduleDir + mc.NAME + pathSeparator); os.IsNotExist(err) {
			listArgs = []string{"clone", mc.EXE.SRC}
			action = "Downloaded"
			path = moduleDir
		} else {
			listArgs = []string{"pull"}
			action = "Update"
			path = moduleDir + mc.NAME + pathSeparator
		}

		cmd := exec.Command("git", listArgs...)
		cmd.Dir = path
		out, err := cmd.Output()
		fmt.Println("GO-WOXY Core -", action, "mod :", mc, "-", string(out), err)

		mc.EXE.BIN = moduleDir + mc.NAME + pathSeparator
		mc.STATE = Downloaded
	} else {
		log.Println("GO-WOXY Core - Error trying to download/update module while running. Stop it before")
	}
}

//GetLog - GetLog from Module
func (mc *ModuleConfig) GetLog() string {
	file, err := os.Open(GetManager().GetConfig().MODDIR + mc.NAME + "/log.log")
	if err != nil {
		log.Fatalln("failed reading file :", err)
	}
	b, err := ioutil.ReadAll(file)
	return string(b)
}

//GetPerf - GetPerf from Module
func (mc *ModuleConfig) GetPerf() (float64, float32) {
	var p, err = process.NewProcess(int32(mc.pid))
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
	if strings.Contains(mc.TYPES, "resource") {
		sP := ""
		if len(paths[0].FROM) > 1 {
			sP = paths[0].FROM
		}
		r := Route{FROM: sP + "/" + mc.RESOURCEPATH + "*filepath", TO: "/" + mc.RESOURCEPATH + "*filepath"}
		err := mc.Hook(router, r, "GET")
		if err != nil {
			log.Panicln("GO-WOXY Core - Error cannot bind resource at the same address")
		}
	}

	if len(paths) > 0 && len(paths[0].FROM) > 0 {
		for i := range paths {
			err := mc.Hook(router, paths[i], "Any")
			if err != nil {
				return err
			}
		}
	}
	return nil
}

//Hook - Create a binding between module and gin server
func (mc *ModuleConfig) Hook(router *gin.Engine, r Route, typeR string) error {
	routes := router.Routes()
	for i := range routes {
		if routes[i].Path == r.FROM && routes[i].Method == typeR {
			return nil
		}
	}

	if typeR == "" || typeR == "Any" {
		typeR = "GET"
	}
	if len(r.FROM) > 0 {
		if mc.AUTH.ENABLED {
			_, err := os.Stat(".htpasswd")
			if os.IsNotExist(err) {
				log.Panicln("GO-WOXY Core - Hook " + mc.NAME + " : .htpasswd file not found")
			} else {
				if typeR == "Any" {
					typeR = "GET"
				}
				htpasswd := auth.HtpasswdFileProvider(".htpasswd")
				authenticator := auth.NewBasicAuthenticator("Some Realm", htpasswd)
				authorized := router.Group("/", BasicAuth(authenticator))
				authorized.Handle(typeR, r.FROM, ReverseProxy(mc.NAME, r))
			}
		} else if typeR != "Any" {
			router.Handle(typeR, r.FROM, ReverseProxy(mc.NAME, r))
		} else {
			router.Any(r.FROM, ReverseProxy(mc.NAME, r))
		}
		fmt.Println("GO-WOXY Core - Module " + mc.NAME + " Hooked to Go-Proxy Server at - " + r.FROM + " => " + r.TO)
	}
	return nil
}

//Setup - Setup module from config
func (mc *ModuleConfig) Setup(router *gin.Engine, hook bool, modulePath string) error {
	fmt.Println("GO-WOXY Core - Setup mod : ", mc)
	if hook && reflect.DeepEqual(mc.EXE, ModuleExecConfig{}) {
		err := mc.HookAll(router)
		if err != nil {
			log.Println(err)
		}
		mc.STATE = Online
	}

	if !mc.EXE.REMOTE && !reflect.DeepEqual(mc.EXE, ModuleExecConfig{}) {
		if strings.Contains(mc.EXE.SRC, "http") || strings.Contains(mc.EXE.SRC, "git@") {
			mc.Download(modulePath)
		}
		mc.copySecret()
		mc.STATE = Loading
		go mc.Start()
	} // ELSE NO BUILD

	return nil
}

//Start - Start module with config args and auto args
func (mc *ModuleConfig) Start() {
	fmt.Println("GO-WOXY Core - Starting mod : ", mc)

	var platformParam []string
	if runtime.GOOS == "windows" {
		platformParam = []string{"cmd", "/c ", "go", "run", mc.EXE.MAIN, "1>", "log.log", "2>&1"}
	} else {
		platformParam = []string{"/bin/sh", "-c", "go run " + mc.EXE.MAIN + " > log.log 2>&1"}
	}

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

	destination, err := os.Create(mc.EXE.BIN + string(os.PathSeparator) + ".secret")
	if err != nil {
		log.Println("GO-WOXY Core - Error creating mod secret file")
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	if err != nil {
		log.Println("GO-WOXY Core - Error Copying Secret:", err, nBytes)
	}
}

// BasicAuth - Authentication gin middleware
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
		mod := GetManager().GetConfig().MODULES[modName]
		//CHECK IF MODULE IS ONLINE
		if mod.STATE == Online {
			//IF ROOT IS PRESENT REDIRECT TO IT
			if strings.Contains(mod.TYPES, "bind") && mod.BINDING.ROOT != "" {
				c.File(mod.BINDING.ROOT)
				//ELSE IF BINDING IS TYPE **WEB**
			} else if strings.Contains(mod.TYPES, "web") {
				//REVERSE PROXY TO IT
				urlProxy, err := url.Parse(mod.BINDING.PROTOCOL + "://" + mod.BINDING.ADDRESS + ":" + mod.BINDING.PORT + r.TO)
				if err != nil {
					log.Println(err)
				}
				proxy := httputil.NewSingleHostReverseProxy(urlProxy)
				proxy.ServeHTTP(c.Writer, c.Request)
			}
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

/*ModuleConfig - Module configuration */
type ModuleConfig struct {
	AUTH         ModuleAuthConfig
	BINDING      ServerConfig
	COMMANDS     []string
	EXE          ModuleExecConfig
	NAME         string
	pid          int
	PK           string
	RESOURCEPATH string
	STATE        ModuleState
	TYPES        string
	VERSION      int
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

/*ModuleAuthConfig - ModuleConfig Auth configuration*/
type ModuleAuthConfig struct {
	ENABLED bool
	TYPE    string
}

// Route - Route redirection
type Route struct {
	FROM string
	TO   string
}

//ModuleState - ModuleConfig State
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
