package core

import (
	"errors"
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
	"text/template"

	"github.com/Wariie/go-woxy/com"
	auth "github.com/abbot/go-http-auth"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/shirou/gopsutil/process"
)

//FileBind - File bind handler
func FileBind(fileName string, r Route) http.HandlerFunc {
	return func(w http.ResponseWriter, re *http.Request) {
		if fileName != "" {
			http.ServeFile(w, re, fileName)
		} else {
			w.Write([]byte("GO-WOXY Core - Error Bind - " + fileName + " was not found"))
		}
	}
}

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
		log.Println("GO-WOXY Core - Downloading " + mc.NAME)
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
		log.Println("GO-WOXY Core - ", action, "mod :", mc, "-", string(out), err)

		mc.EXE.BIN = moduleDir + mc.NAME + pathSeparator
		mc.STATE = Downloaded
	} else {
		log.Println("GO-WOXY Core - Error trying to download/update module while running. Stop it before")
	}
}

//ErrorHandler -
func ErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	title := "Error"
	//message := "Error"

	data := ErrorPage{
		title:   title,
		code:    400,
		message: err.Error(),
	}

	tmpl := template.Must(template.ParseFiles("./resources/html/loading.html"))
	tmpl.Execute(w, data)
}

//GetLog - GetLog from Module
func (mc *ModuleConfig) GetLog() string {

	//TODO ADD REMOTE COMMAND TO GET LOG
	file, err := os.Open(GetManager().GetConfig().MODDIR + mc.NAME + "/log.log")
	if err != nil {
		log.Fatalln("GO-WOXY Core - Error reading log file :", err)
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

//Handle404Status - Throw err when proxied response status is 404
func Handle404Status(res *http.Response) error {
	if res.StatusCode == 404 {
		return errors.New("404 error from the host")
	}
	return nil
}

//HookAll - Create all binding between module config address and router server
func (mc *ModuleConfig) HookAll(router *mux.Router) error {
	paths := mc.BINDING.PATH
	var err error

	for i := range paths {
		err = mc.Hook(router, paths[i], "Any")
		if err != nil {
			log.Println("GO-WOXY Core - Error during module path hooking : " + err.Error())
			return err
		}
	}
	return err
}

//Hook - Create a binding between module and router server
func (mc *ModuleConfig) Hook(router *mux.Router, r Route, typeR string) error {
	if len(r.FROM) > 0 {
		var handler http.Handler
		if mc.AUTH.ENABLED {
			_, err := os.Stat(".htpasswd")
			if os.IsNotExist(err) {
				log.Panicln("GO-WOXY Core - Hook " + mc.NAME + " : .htpasswd file not found")
			} else {
				htpasswd := auth.HtpasswdFileProvider(".htpasswd")
				authenticator := auth.NewBasicAuthenticator("guilhem-mateo.fr mod-manager", htpasswd)
				handler = ReverseProxyAuth(authenticator, mc.NAME, r)
			}
		} else if strings.Contains(mc.TYPES, "bind") {
			handler = FileBind(mc.BINDING.ROOT, r)
		} else {
			handler = ReverseProxy(mc.NAME, r)
		}

		if handler != nil {
			router.PathPrefix(r.FROM).Handler(handlers.CombinedLoggingHandler(GetManager().GetAccessLogFileWriter(), handler))
			log.Println("GO-WOXY Core - Module " + mc.NAME + " - Route created : " + r.FROM + " > " + r.TO)
		} else {
			log.Println("GO-WOXY Core - Error hooking module " + mc.NAME + " - Route : " + r.FROM + " > " + r.TO)
		}
	}

	return nil
}

// ReverseProxyAuth - Authentication middleware
func ReverseProxyAuth(a *auth.BasicAuth, modName string, r Route) http.HandlerFunc {
	return func(w http.ResponseWriter, re *http.Request) {
		user := a.CheckAuth(re)
		if user == "" {
			w.Header().Add("WWW-Authenticate", "Basic realm="+strconv.Quote(a.Realm))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("user", user)
		ReverseProxy(modName, r)(w, re)
	}
}

//ReverseProxy - reverse proxy for mod
func ReverseProxy(modName string, r Route) http.HandlerFunc {
	return func(w http.ResponseWriter, re *http.Request) {
		mod := GetManager().GetModule(modName)

		//CHECK IF MODULE IS ONLINE
		if mod.STATE == Online {

			//IF ROOT IS PRESENT REDIRECT TO IT
			if strings.Contains(mod.TYPES, "bind") && mod.BINDING.ROOT != "" {
				http.ServeFile(w, re, mod.BINDING.ROOT)
				//ELSE IF BINDING IS TYPE **REVERSE**
			} else if strings.Contains(mod.TYPES, "reverse") {

				path := re.URL.Path
				if r.FROM != r.TO {
					if r.FROM != "/" {
						i := strings.Index(path, r.FROM)
						path = path[i+len(r.FROM):]
					}

					if r.TO != "/" && len(r.TO) > 1 && !strings.Contains(path, r.TO) {
						path = r.TO + path
					}
				}

				//BUILD URL PROXY
				urlProxy, err := url.Parse(mod.BINDING.PROTOCOL + "://" + mod.BINDING.ADDRESS + ":" + mod.BINDING.PORT + path)
				if err != nil {
					log.Println(err) //TODO ERROR HANDLING
				}

				//TODO ADD CUSTOM HEADERS HERE

				//SETUP REVERSE PROXY DIRECTOR
				proxy := httputil.NewSingleHostReverseProxy(urlProxy)
				proxy.Director = func(req *http.Request) {

					req.URL.Scheme = urlProxy.Scheme
					req.Host = urlProxy.Host
					req.URL.Host = urlProxy.Host
					req.URL.Path = urlProxy.Path

					if _, ok := req.Header["User-Agent"]; !ok {
						req.Header.Set("User-Agent", "")
					}
				}
				proxy.ErrorHandler = ErrorHandler
				proxy.ModifyResponse = Handle404Status
				proxy.ServeHTTP(w, re)
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

			data := ErrorPage{
				title:   title,
				code:    code,
				message: message,
			}

			tmpl := template.Must(template.ParseFiles("./resources/html/loading.html"))
			tmpl.Execute(w, data)
		}
	}
}

//Setup - Setup module from config
func (mc *ModuleConfig) Setup(router *mux.Router, hook bool, modulePath string) error {
	log.Println("GO-WOXY Core - Setup mod : ", mc)
	if hook && reflect.DeepEqual(mc.EXE, ModuleExecConfig{}) {
		err := mc.HookAll(router)
		if err != nil {
			log.Println(err)
		}
		mc.STATE = Online
	}

	//IF CONTAINS EXE CONFIG && NOT REMOTE
	if !mc.EXE.REMOTE && !reflect.DeepEqual(mc.EXE, ModuleExecConfig{}) {
		if strings.Contains(mc.EXE.SRC, "http") || strings.Contains(mc.EXE.SRC, "git@") {
			mc.Download(modulePath)
		}
		mc.copySecret()
		mc.STATE = Loading
		go mc.Start()
	}

	return nil
}

//Start - Start module with config args and auto args
func (mc *ModuleConfig) Start() {
	log.Println("GO-WOXY Core - Starting mod : ", mc)

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
		log.Println("GO-WOXY Core - Error reading generated secret file : ", err)
	}
	defer source.Close()

	destination, err := os.Create(mc.EXE.BIN + string(os.PathSeparator) + ".secret")
	if err != nil {
		log.Println("GO-WOXY Core - Error creating mod secret file : ", err)
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	if err != nil {
		log.Println("GO-WOXY Core - Error Copying Secret : ", err, nBytes)
	}
}

//ErrorPage - Content description for go-woxy error page
type ErrorPage struct {
	title   string
	code    int
	message string
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

//Route - Route redirection
type Route struct {
	FROM string
	TO   string
}

//ModuleState - ModuleConfig State
type ModuleState string

//ModuleState list
const (
	Unknown    ModuleState = "UNKNOWN"
	Loading    ModuleState = "LOADING"
	Online     ModuleState = "ONLINE"
	Stopped    ModuleState = "STOPPED"
	Downloaded ModuleState = "DOWNLOADED"
	Error      ModuleState = "ERROR"
	Failed     ModuleState = "FAILED"
)
