package core

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Wariie/go-woxy/com"
	auth "github.com/abbot/go-http-auth"
	"github.com/sirupsen/logrus"
)

//Core - GO-WOXY Core Server
type Core struct {
	cp          *CommandProcessorImpl
	config      *Config
	loggers     map[string]*logrus.Logger
	modulesList []ModuleConfig
	mux         sync.Mutex
	router      *Router
	s           *Supervisor
	server      *HttpServer
	roles       []Role
}

//GetConfig - Get go-woxy config
func (core *Core) GetConfig() Config {
	core.mux.Lock()
	defer core.mux.Unlock()
	return *core.config
}

//GetServer - Get server
func (core *Core) GetServer() *HttpServer {
	core.mux.Lock()
	defer core.mux.Unlock()
	return core.server
}

//SetServer - Set server
func (core *Core) SetServer(s *HttpServer) {
	core.mux.Lock()
	defer core.mux.Unlock()
	core.server = s
}

//GetCommandProcessor - Get CommandProcessor
func (core *Core) GetCommandProcessor() *CommandProcessorImpl {
	core.mux.Lock()
	defer core.mux.Unlock()
	return core.cp
}

//GetSupervisor - Get module supervisor
func (core *Core) GetSupervisor() *Supervisor {
	core.mux.Lock()
	defer core.mux.Unlock()
	return core.s
}

//GetModule - Get module reference from core list module
func (core *Core) GetModule(name string) *ModuleConfig {
	core.mux.Lock()
	defer core.mux.Unlock()

	for _, m := range core.modulesList {
		if m.NAME == name {
			return &m
		}
	}
	return &ModuleConfig{}
}

//HookAll - Create all binding between module config address and router server
func (core *Core) HookAll(mc *ModuleConfig) {
	routes := mc.BINDING.PATH
	var err error

	for _, route := range routes {
		err = core.Hook(mc, route)
		if err != nil {
			log.Println("GO-WOXY Core - Error hooking", mc.NAME, "- Route", route.FROM, " > ", route.TO, ":", err.Error())
		}
	}
}

//Hook - Create a binding between module and router server
func (core *Core) Hook(mc *ModuleConfig, r Route) error {
	var err error
	if len(r.FROM) > 0 {
		var handler HandlerFunc
		if mc.AUTH.ENABLED {
			_, err = os.Stat(".htpasswd")
			if os.IsNotExist(err) {
				err = errors.New(".htpasswd file not found")
			} else {
				htpasswd := auth.HtpasswdFileProvider(".htpasswd")
				//TODO HANDLE PARAMETERS
				authenticator := auth.NewBasicAuthenticator("guilhem-mateo.fr mod-manager", htpasswd)
				handler = ReverseProxyAuth(authenticator, mc.NAME, r)
			}
		} else if strings.Contains(mc.TYPES, "bind") {
			handler = FileBind(mc.BINDING.ROOT, r)
		} else {
			handler = ReverseProxy()
		}

		if handler != nil {
			core.router.Handle(r.FROM, handler, mc, &r)
			log.Println("GO-WOXY Core - Module " + mc.NAME + " - Route created : " + r.FROM + " > " + r.TO)
		} else {
			err = errors.New("no handler found with this configuration")
		}
	}

	return err
}

//SaveModuleChanges - Thread safe way to edit Module state
func (core *Core) SaveModuleChanges(mc *ModuleConfig) {
	core.mux.Lock()
	defer core.mux.Unlock()
	for i, m := range core.modulesList {
		if m.NAME == mc.NAME {
			core.modulesList[i] = *mc
			return
		}
	}
}

//SearchModWithHash - Thread safe way to get module with his hash
func (core *Core) SearchModWithHash(hash string) *ModuleConfig {
	core.mux.Lock()
	defer core.mux.Unlock()
	for _, m := range core.modulesList {
		if m.PK == hash {
			return &m
		}
	}
	return &ModuleConfig{NAME: "error"}
}

//Setup - Setup module from config
func (core *Core) Setup(mc ModuleConfig, hook bool, modulePath string) (*ModuleConfig, error) {
	log.Println("GO-WOXY Core - Setup mod : ", mc)
	if hook && reflect.DeepEqual(mc.EXE, ModuleExecConfig{}) {
		core.HookAll(&mc)
		mc.STATE = Online
	}

	//IF CONTAINS EXE CONFIG && NOT REMOTE
	if !reflect.DeepEqual(mc.EXE, ModuleExecConfig{}) {
		mc.generateAPIKey()
		if !mc.EXE.REMOTE && (strings.Contains(mc.EXE.SRC, "http") || strings.Contains(mc.EXE.SRC, "git@")) {
			mc.Download(modulePath)
			mc.copyAPIKey()
		}
		mc.STATE = Loading
	}
	return &mc, nil
}

func (core *Core) launchServer() {
	log.Println("GO-WOXY Core - Starting")

	//AUTHENTICATION & COMMAND ENDPOINT
	core.router.DefaultRoute = error404()
	core.router.Handle("/connect", core.connect(), nil, nil)
	core.router.Handle("/cmd", core.command(), nil, nil)

	core.configAndServe()
}

func (core *Core) configAndServe() {
	path := ""
	server := core.config.SERVER

	if len(server.PATH) > 0 {
		path = server.PATH[0].FROM
	}
	log.Println("GO-WOXY Core - Serving at " + server.PROTOCOL + "://" + server.ADDRESS + ":" + server.PORT + path)

	var s = HttpServer{
		Server: http.Server{
			Addr:         server.ADDRESS + ":" + server.PORT + path,
			Handler:      core.router,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
		shutdownReq: make(chan bool),
	}

	var listener net.Listener
	var err error

	//CHECK FOR CERTIFICATE TO TRY TLS CONFIG
	if server.CERT != "" && server.CERT_KEY != "" {
		tlsConfig, err := core.getTLSConfig()
		if err != nil {
			log.Fatalln("GO-WOXY Core - Error creating tls config : ", err)
		}

		listener, err = tls.Listen("tcp", server.ADDRESS+":"+server.PORT+path, tlsConfig)

		if err != nil {
			log.Fatal("GO-WOXY Core - ", err)
		}

	} else {
		listener, err = net.Listen("tcp", server.ADDRESS+":"+server.PORT+path)
		if err != nil {
			log.Fatal("GO-WOXY Core -", err)
		}
	}

	core.SetServer(&s)

	done := make(chan bool)
	go func() {
		err := s.Serve(listener)
		if err != nil {
			log.Printf("GO-WOXY Core - %v", err)
		}
		done <- true
	}()

	//wait shutdown
	s.WaitShutdown()

	<-done
	log.Printf("GO-WOXY Core - Stopped")
}

func (core *Core) getTLSConfig() (*tls.Config, error) {
	server := core.config.SERVER
	cer, err := tls.LoadX509KeyPair(server.CERT, server.CERT_KEY)
	if err != nil {
		return &tls.Config{}, err
	}
	return &tls.Config{Certificates: []tls.Certificate{cer}}, nil
}

func (core *Core) loadModules() {
	//INIT MODULE DIRECTORY
	wd, err := os.Getwd()

	modDirName := core.config.MODDIR

	if err != nil {
		log.Fatalln("GO-WOXY Core - Error opening $pwd : ", err)
	}

	err = os.Mkdir(wd+string(os.PathSeparator)+modDirName, os.ModeDir)
	if err != nil && os.IsNotExist(err) {
		log.Fatalln("GO-WOXY Core - Error creating mods folder : ", err)
	}

	core.mux.Lock()
	defer core.mux.Unlock()
	for i, m := range core.modulesList {
		i := i
		m := m
		mN, err := core.Setup(m, true, modDirName)
		core.modulesList[i] = *mN
		if err != nil {
			log.Fatalln("GO-WOXY Core - Error setup module ", m.NAME, " : ", err)
		}
	}

	//ADD HUB MODULE FOR COMMAND GESTURE
	core.modulesList = append(core.modulesList, ModuleConfig{NAME: "hub", PK: "hub"})
}

func (core *Core) startModules() {
	time.Sleep(time.Second * 1)

	core.mux.Lock()
	defer core.mux.Unlock()

	for i, m := range core.modulesList {
		if len(m.EXE.BIN) > 0 {

			//START MODULE
			m.Start()

			//ADD IT TO SUPERVISOR IF SUPERVISED
			if m.EXE.SUPERVISED {
				core.s.Add(m.NAME)
			}

			//SAVE CHANGES
			core.modulesList[i] = m
		}
	}
}

func (core *Core) init() {

	core.mux.Lock()
	defer core.mux.Unlock()

	//Load ModuleConfigs from loaded config
	for _, mod := range core.config.MODULES {
		core.modulesList = append(core.modulesList, mod)
	}

	core.config.generateSecret()

	if len(core.config.ACCESSLOGFILE) == 0 {
		core.config.ACCESSLOGFILE = "access.log"
	}

	//Setup go-woxy log files
	core.initLogs()

	//Setup server router with error handling page
	router := NewRouter(error404()) //Custom Http Router
	router.Middlewares = append(router.Middlewares, core.logMiddleware())

	//Setup CommandProcessor
	cp := CommandProcessorImpl{}
	cp.Init()
	core.cp = &cp
	core.router = router
}

func (core *Core) initLogs() {

	//Open default system access log file
	coreLogFile, err := os.OpenFile("core.log", os.O_APPEND|os.O_CREATE|os.O_RDWR, os.ModePerm)
	if err != nil {
		log.Fatalln("GO-WOXY Core - Error opening access log file " + core.config.ACCESSLOGFILE + " : " + err.Error())
	}

	//Open default access log file
	accessLogFile, err := os.OpenFile(core.config.ACCESSLOGFILE, os.O_APPEND|os.O_CREATE|os.O_RDWR, os.ModePerm)
	if err != nil {
		log.Fatalln("GO-WOXY Core - Error opening access log file " + core.config.ACCESSLOGFILE + " : " + err.Error())
	}

	//Init logrus
	var coreLogger = logrus.Logger{
		Out:       coreLogFile,
		Formatter: new(logrus.TextFormatter),
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.DebugLevel,
	}

	//init go-woxy loggers map
	// - accessLogFile - default log file for module
	// - coreLogFile - system access log file
	// * 1/module
	core.loggers = make(map[string]*logrus.Logger, len(core.modulesList)+2)

	coreLogger.SetFormatter(&logrus.TextFormatter{})
	core.loggers["core"] = &coreLogger

	accessLogger := *&coreLogger
	accessLogger.SetOutput(accessLogFile)
	core.loggers["access"] = &accessLogger

	//CREATE CUSTOM MODULE LOGGERS
	for _, m := range core.modulesList {
		logger := *&accessLogger
		var logFile *os.File = accessLogFile
		if m.LOG.IsEnabled() && m.LOG.Path != "default" {
			path := m.LOG.Path + m.LOG.File
			logFile, err = os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_RDWR, os.ModePerm)
			if err != nil {
				log.Println("GO-WOXY Core - Error opening access log file " + path + " : " + err.Error())
			}
		}

		if logFile != nil && err == nil {
			logger.SetOutput(logFile)
			core.loggers[m.NAME] = &logger
		}
	}
}

func (core *Core) logMiddleware() MiddlewareFunc {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx *Context) {
			requestLog := fmt.Sprintf("%s %s %s",
				ctx.Request.Method,
				ctx.URL.Path,
				ctx.Request.Proto,
			)
			var routedToLog string

			var logger *logrus.Logger
			if ctx.URL.Path == "/cmd" || ctx.URL.Path == "/connect" {
				//TODO ADD LOGGER CONSTS
				logger = core.GetLogger("core")
				routedToLog = "internally"
			} else {
				logger = core.GetLogger(ctx.ModuleConfig.NAME)
				if ctx.ModuleConfig != nil {
					routedToLog = ctx.ModuleConfig.NAME
				} else {
					routedToLog = "NOT FOUND"
				}
			}
			logger.WithFields(logrus.Fields{
				"from":    ctx.RemoteAddr,
				"request": requestLog,
				"to":      routedToLog,
			}).Info("routed")
			next.Handle(ctx)
		})
	}
}

/*LoadConfigFromPath - Load config file from path */
func (core *Core) loadConfigFromPath(configPath string) {
	cfg := Config{}
	cfg.Load(configPath)
	core.config = &cfg
}

//GoWoxy - start core server
func (core *Core) GoWoxy(configPath string) {

	//Load Config
	core.loadConfigFromPath(configPath)
	core.showMotd()

	//Init Go-Woxy core
	core.init()

	// START MODULE SUPERVISOR
	core.initSupervisor()

	// SETUP MODULES
	core.loadModules()

	// BATCH START MODULES
	go core.startModules()

	// START SERVER WHERE MODULES WILL REGISTER
	core.launchServer()
}

func (core *Core) initSupervisor() {
	core.s = &Supervisor{}
	core.s.core = core
	go core.s.Supervise()
}

func (core *Core) showMotd() {
	fmt.Println(" -------------------- Go-Woxy - V 0.0.1 -------------------- ")
	fmt.Println(core.config.GetMotdFileContent())
	fmt.Println("------------------------------------------------------------ ")
}

func (core *Core) registerModule(m *ModuleConfig, cr *com.ConnexionRequest) {

	pid, err := strconv.Atoi(cr.Pid)
	if err != nil {
		log.Println("GO-WOXY Core - Error reading PID :", err)
	}

	m.pid = pid
	m.PK = cr.ModHash
	m.COMMANDS = cr.CustomCommands

	if len(m.BINDING.PORT) == 0 || len(cr.Port) > 0 {
		m.BINDING.PORT = cr.Port
	}

	core.HookAll(m)
}

func (core *Core) connect() HandlerFunc {
	return HandlerFunc(func(ctx *Context) {

		//READ Body and try to found ConnexionRequest
		var cr com.ConnexionRequest
		buf := new(bytes.Buffer)
		buf.ReadFrom(ctx.Body)
		cr.Decode(buf.Bytes())

		var modC *ModuleConfig = core.GetModule(cr.Name)

		var resultW []byte
		if modC == nil || len(modC.NAME) <= 0 {
			//ERROR DURING THE BODY PARSING
			errMsg := "Error reading ConnexionRequest"
			log.Println("GO-WOXY Core - " + errMsg)
			resultW = []byte(errMsg) //TODO BETTER RESPONSE
		} else {

			log.Println("GO-WOXY Core - Module connecting with name '" + modC.NAME + "'")

			//GET THE REMOTE HOST ADDRESS IF HOST IS EMPTY
			if !modC.EXE.REMOTE {
				modC.BINDING.ADDRESS = strings.Split(ctx.Host, ":")[0]
			}

			//CHECK SECRET FOR AUTH
			rs := modC.APIKeyMatch(cr.Secret)
			if rs && cr.ModHash != "" {
				modC.STATE = Online
				core.registerModule(modC, &cr)
			} else {
				modC.STATE = Failed
			}

			core.SaveModuleChanges(modC)

			//SEND RESPONSE
			result := strconv.FormatBool(rs)
			log.Println("GO-WOXY Core - Module", modC.NAME, "connecting - result :", result)

			cr.State = result
			resultW = cr.Encode()
		}
		i, err := ctx.ByteText(200, resultW)
		if err != nil {
			log.Println("GO-WOXY Core - Module", modC.NAME, " failed to respond :", err.Error(), " bytes : ", i)
		}

	})
}

// Command - Access point to handle module commands
func (core *Core) command() HandlerFunc {
	return HandlerFunc(func(ctx *Context) {
		t, b := com.GetCustomRequestType(ctx.Request)

		from := ctx.Request.RemoteAddr
		response := ""
		action := ""

		// CHECK ERROR DURING READING DATA
		if t["error"] == "error" {
			response = "Error reading Request"
		} else if t["Hash"] != "" {

			//GET MOD WITH HASH
			mc := core.SearchModWithHash(t["Hash"])

			if mc.NAME == "error" {
				response = "Error module not found"
				//TODO HANDLE HUB COMMANDS AUTHENTICATION
			} else if mc.NAME == "hub" || mc.APIKeyMatch(t["Secret"]) {
				action += "To " + mc.NAME + " - "

				//PROCESS REQUEST
				switch t["Type"] {
				case "Command":
					var cr com.CommandRequest
					cr.Decode(b)
					var c interface{} = &cr
					p := (c).(com.Request)
					res, e := core.GetCommandProcessor().Run(cr.Command, core, &p, mc, "")
					response += res
					if e != nil {
						response = response + " " + e.Error()
					}
					action += "Command [ " + cr.Command + " ]"
				}
				//core.config.MODULES.Set(mc.NAME, mc)
			} else {
				response = "Secret not matching with server"
			}

			core.SaveModuleChanges(mc)
		} else {
			if len(t["Hash"]) == 0 {
				response = "Empty Hash : Try to start module"
			} else {
				response = "Unknown error"
			}
		}

		action += " - Result : " + response

		//LOG COMMAND RESULT
		log.Println("GO-WOXY Core - From", from, action)

		ctx.ByteText(200, []byte(response))
	})
}

//HttpServer -
type HttpServer struct {
	http.Server
	shutdownReq chan bool
	reqCount    uint32
}

//WaitShutdown - Wait server to shutdown correctly
func (s *HttpServer) WaitShutdown() {
	irqSig := make(chan os.Signal, 1)
	signal.Notify(irqSig, syscall.SIGINT, syscall.SIGTERM)

	//Wait interrupt or shutdown request through /shutdown
	select {
	case sig := <-irqSig:
		log.Printf("GO-WOXY Core - Shutdown request (signal: %v)", sig)
	case sig := <-s.shutdownReq:
		log.Printf("GO-WOXY Core - Shutdown request (/shutdown %v)", sig)
	}

	log.Printf("GO-WOXY Core - Stoping http server ...")

	//Create shutdown context with 10 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	//shutdown the server
	err := s.Shutdown(ctx)
	if err != nil {
		log.Printf("Shutdown request error: %v", err)
	}
}
