package core

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"text/template"
	"time"

	_ "net/http/pprof"

	"github.com/Wariie/go-woxy/com"
	"github.com/Wariie/go-woxy/tools"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

//Core - GO-WOXY Core Server
type Core struct {
	mux           sync.Mutex
	config        Config
	router        *mux.Router
	cp            *CommandProcessorImpl
	s             *Supervisor
	server        *HttpServer
	roles         []Role
	accessLogFile *os.File
}

func (core *Core) GetConfig() Config {
	core.mux.Lock()
	defer core.mux.Unlock()
	return core.config
}

func (core *Core) SetConfig(c Config) {
	core.mux.Lock()
	defer core.mux.Unlock()
	core.config = c
}

func (core *Core) GetRouter() *mux.Router {
	core.mux.Lock()
	defer core.mux.Unlock()
	return core.router
}

func (core *Core) SetRouter(r *mux.Router) {
	core.mux.Lock()
	defer core.mux.Unlock()
	core.router = r
}

func (core *Core) GetServer() *HttpServer {
	core.mux.Lock()
	defer core.mux.Unlock()
	return core.server
}

func (core *Core) SetServer(s *HttpServer) {
	core.mux.Lock()
	defer core.mux.Unlock()
	core.server = s
}

func (core *Core) GetCommandProcessor() *CommandProcessorImpl {
	core.mux.Lock()
	defer core.mux.Unlock()
	return core.cp
}

func (core *Core) SetCommandProcessor(cp *CommandProcessorImpl) {
	core.mux.Lock()
	defer core.mux.Unlock()
	core.cp = cp
}

func (core *Core) SetSupervisor(s *Supervisor) {
	core.mux.Lock()
	defer core.mux.Unlock()
	core.s = s
}

func (core *Core) GetSupervisor() *Supervisor {
	core.mux.Lock()
	defer core.mux.Unlock()
	return core.s
}

func (core *Core) AddModuleToSupervisor(mc *ModuleConfig) {
	core.mux.Lock()
	defer core.mux.Unlock()
	core.s.Add(mc.NAME)
}

func (core *Core) GetModule(name string) *ModuleConfig {
	core.mux.Lock()
	defer core.mux.Unlock()

	for _, m := range core.config.modulesList {
		if m.NAME == name {
			return m
		}
	}
	return &ModuleConfig{}
}

func (core *Core) SaveModuleChanges(mc *ModuleConfig) {
	core.mux.Lock()
	defer core.mux.Unlock()
	for _, m := range core.config.modulesList {
		if m.NAME == mc.NAME {
			m = mc
			return
		}
	}
}

func (core *Core) SearchModWithHash(hash string) *ModuleConfig {
	core.mux.Lock()
	defer core.mux.Unlock()
	for _, m := range core.config.modulesList {
		if m.PK == hash {
			return m
		}
	}
	return &ModuleConfig{NAME: "error"}
}

func (core *Core) GetAccessLogFileWriter() io.Writer {
	return bufio.NewWriter(core.accessLogFile)
}

func (core *Core) SetAccessLogFile(accesslogfile *os.File) {
	core.accessLogFile = accesslogfile
}

//TODO DELETE AND ADD API KEY HANDLING
func (core *Core) generateSecret() {
	if len(core.config.SECRET) == 0 {
		b := []byte(tools.String(64))
		err := ioutil.WriteFile(".secret", b, 0644)
		if err != nil {
			log.Fatalln("GO-WOXY Core - Error creating secret file : ", err)
		}
		h := sha256.New()
		h.Write(b)
		core.config.SECRET = base64.URLEncoding.EncodeToString(h.Sum(nil))
	}
}

func (core *Core) launchServer() {

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	log.Println("GO-WOXY Core - Starting")
	//AUTHENTICATION & COMMAND ENDPOINT

	//SWITCH THIS TO INTERNAL LOGGING INSTEAD OF ACCESS LOGGING ?
	core.router.NotFoundHandler = handlers.CombinedLoggingHandler(core.accessLogFile, http.HandlerFunc(core.error404))
	core.router.PathPrefix("/connect").Handler(handlers.CombinedLoggingHandler(core.accessLogFile, core.connect()))
	core.router.PathPrefix("/cmd").Handler(handlers.CombinedLoggingHandler(core.accessLogFile, core.command()))

	core.configAndServe(core.router)
}

func (core *Core) error404(w http.ResponseWriter, r *http.Request) {
	fp := path.Join("resources/html", "404.html")
	tmpl, err := template.ParseFiles(fp)
	if err != nil {
		log.Println("GO-WOXY Core - Error 404 template Not Found")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(404)
	if err := tmpl.Execute(w, nil); err != nil {
		log.Println("GO-WOXY Core - Error executing 404 template")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		log.Println("GO-WOXY Core - 404 Not Found")
	}
}

func (core *Core) configAndServe(router *mux.Router) {
	path := ""
	server := core.config.SERVER

	if len(server.PATH) > 0 {
		path = server.PATH[0].FROM
	}
	log.Println("GO-WOXY Core - Serving at " + server.PROTOCOL + "://" + server.ADDRESS + ":" + server.PORT + path)

	var s = HttpServer{
		Server: http.Server{
			Addr:         server.ADDRESS + ":" + server.PORT + path,
			Handler:      router,
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
			log.Fatal(err)
		}

	} else {
		listener, err = net.Listen("tcp", server.ADDRESS+":"+server.PORT+path)
		if err != nil {
			log.Fatal(err)
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
	if err != nil {
		errMsg := "GO-WOXY Core - Error creating mods folder : "
		if os.IsNotExist(err) {
			log.Fatalln(errMsg, err)
		} else if os.IsExist(err) {
			log.Println(errMsg, err)
		}
	}

	for _, m := range core.config.modulesList {
		err := core.Setup(m, true, modDirName)
		if err != nil {
			log.Fatalln("GO-WOXY Core - Error setup module ", m.NAME, " : ", err)
		}

		//modulesList.Set(m.NAME, m)
	}

	//ADD HUB MODULE FOR COMMAND GESTURE
	core.config.modulesList = append(core.config.modulesList, &ModuleConfig{NAME: "hub", PK: "hub"})
}

func (core *Core) init() {

	//TODO HANDLE TEMPLATE SOURCE DIR
	//router.LoadHTMLGlob("." + string(os.PathSeparator) + config.RESOURCEDIR + "*" + string(os.PathSeparator) + "*")
	core.generateSecret()

	if len(core.config.ACCESSLOGFILE) == 0 {
		core.config.ACCESSLOGFILE = "access.log"
	}

	//ACCESS LOGGING
	f, err := os.OpenFile(core.config.ACCESSLOGFILE, os.O_APPEND|os.O_CREATE|os.O_RDWR, os.ModePerm)
	if err != nil {
		log.Fatalln("GO-WOXY Core - Error opening access log file " + core.config.ACCESSLOGFILE + " : " + err.Error())
	} else {
		core.accessLogFile = f
	}

	router := mux.NewRouter()
	//router.Use()

	cp := CommandProcessorImpl{}
	cp.Init()
	core.cp = &cp
	core.router = router
}

/*LoadConfigFromPath - Load config file from path */
func (core *Core) loadConfigFromPath(configPath string) {
	core.config = Config{}
	core.config.Load(configPath)
}

//LaunchCore - start core server
func (core *Core) GoWoxy(configPath string) {

	//Load Config
	core.loadConfigFromPath(configPath)
	core.showMotd()

	//Init Go-Woxy core
	core.init()

	// START MODULE SUPERVISOR
	core.initSupervisor()

	// STEP 4 LOAD MODULES
	go core.loadModules()

	// STEP 5 START SERVER WHERE MODULES WILL REGISTER
	core.launchServer()
}

func (core *Core) initSupervisor() {
	core.s = &Supervisor{}
	core.s.core = core
	go core.s.Supervise()
}

func (core *Core) showMotd() {
	motd := core.config.GetMotdFileContent()
	//motd := config.GetMotd()
	log.Println(" -------------------- Go-Woxy - V 0.0.1 -------------------- ")
	log.Println(motd)
	log.Println("------------------------------------------------------------ ")
}

func (core *Core) registerModule(m *ModuleConfig, cr *com.ConnexionRequest) bool {
	tm := m

	pid, err := strconv.Atoi(cr.Pid)
	if err != nil {
		log.Println("GO-WOXY Core - Error reading PID :", err)
	}

	m.pid = pid
	m.PK = cr.ModHash
	m.COMMANDS = cr.CustomCommands
	m.STATE = Online
	m.RESOURCEPATH = cr.ResourcePath

	if len(m.RESOURCEPATH) == 0 {
		m.RESOURCEPATH = "resources/"
	}

	if len(m.BINDING.PORT) == 0 || len(cr.Port) > 0 {
		m.BINDING.PORT = cr.Port
	}

	var crr com.CommandRequest
	crr.Generate("Ping", m.PK, m.NAME, core.config.SECRET)
	var c interface{} = &crr
	p := (c).(com.Request)

	time.Sleep(time.Second * 10)

	//RETRY 15 TIME TO CHECK MODULE COME ONLINE
	try := 0
	r := false
	for {
		res, e := core.GetCommandProcessor().Run("Ping", core, &p, m, "")
		log.Print(res, e)

		if res != "" && err == nil {
			r = true
			break
		} else if try > 15 {
			break
		}
		try++
		time.Sleep(time.Second * 1)
	}

	if !r {
		tm.STATE = Failed
		m = tm
	} else {
		err = core.HookAll(m)
		if err == nil && m.EXE.SUPERVISED {
			core.AddModuleToSupervisor(m)
		} else if err != nil {
			log.Println("Go-WOXY Core - Error trying to hook module", m.NAME)
		}
	}

	//TODO CHECK IF WORK WITHTOUT IT
	//core.config.MODULES.Set(m.NAME, m)
	return r
}

func (core *Core) connect() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		//READ Body and try to found ConnexionRequest
		var cr com.ConnexionRequest
		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		cr.Decode(buf.Bytes())

		//GET THE MODULE TARGET
		var modC *ModuleConfig
		for _, m := range core.config.modulesList {
			if m.NAME == cr.Name {
				modC = m
				break
			}
		}

		var resultW []byte
		if modC != nil && len(modC.NAME) > 0 {
			//ERROR DURING THE BODY PARSING
			errMsg := "Error reading ConnexionRequest"
			log.Println("GO-WOXY Core - " + errMsg)
			resultW = []byte(errMsg)
		} else {

			//GET THE REMOTE HOST ADDRESS IF HOST IS EMPTY
			if !modC.EXE.REMOTE {
				modC.BINDING.ADDRESS = strings.Split(r.Host, ":")[0]
			}

			//TODO SET API KEY MECANISM
			//cr.Secret --> API KEY corresponding

			//CHECK SECRET FOR AUTH
			rs := core.hashMatchSecretHash(cr.Secret)
			if rs && cr.ModHash != "" {
				go core.registerModule(modC, &cr)
			} else {
				modC.STATE = Failed
			}

			//SEND RESPONSE
			result := strconv.FormatBool(rs)
			log.Println("GO-WOXY Core - Module", modC.NAME, "connecting - result :", result)

			cr.State = result
			resultW = cr.Encode()
		}
		i, err := w.Write(resultW)
		if err != nil {
			log.Println("GO-WOXY Core - Module", modC.NAME, " failed to respond :", err.Error(), " bytes : ", i)
		}

		//core.config.MODULES.Set(modC.NAME, modC)
	}
}

func (core *Core) hashMatchSecretHash(hash string) bool {
	r := strings.Trim(hash, "\n\t") == strings.Trim(core.config.SECRET, "\n\t")
	return r
}

// Command - Access point to handle module commands
func (core *Core) command() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t, b := com.GetCustomRequestType(r)

		from := r.RemoteAddr
		response := ""
		action := ""

		//CHECK AUTH ( TODO API KEY)
		rs := t["Secret"] == core.config.SECRET

		// CHECK ERROR DURING READING DATA
		if t["error"] == "error" {
			response = "Error reading Request"
		} else if t["Hash"] != "" && rs {

			//GET MOD WITH HASH
			mc := core.SearchModWithHash(t["Hash"])

			if mc.NAME == "error" {
				response = "Error module not found"
			} else {
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
			}
		} else {
			if len(t["Hash"]) == 0 {
				response = "Empty Hash : Try to start module"
			} else if !rs {
				response = "Secret not matching with server"
			} else {
				response = "Unknown error"
			}
		}

		action += " - Result : " + response

		//LOG COMMAND RESULT
		log.Println("GO-WOXY Core - From", from, ':', action)

		w.WriteHeader(200)
		w.Write([]byte(response))
	}
}

type HttpServer struct {
	http.Server
	shutdownReq chan bool
	reqCount    uint32
}

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
