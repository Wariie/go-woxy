package core

import (
	"bytes"
	"log"
	"net/http"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/Wariie/go-woxy/com"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func launchServer() {
	log.Println("GO-WOXY Core - Starting")
	//AUTHENTICATION & COMMAND ENDPOINT

	//SWITCH THIS TO INTERNAL LOGGING INSTEAD OF ACCESS LOGGING ?
	GetManager().GetRouter().NotFoundHandler = handlers.CombinedLoggingHandler(GetManager().GetAccessLogFileWriter(), http.HandlerFunc(error404))
	GetManager().GetRouter().PathPrefix("/connect").Handler(handlers.CombinedLoggingHandler(GetManager().GetAccessLogFileWriter(), connect()))
	GetManager().GetRouter().PathPrefix("/cmd").Handler(handlers.CombinedLoggingHandler(GetManager().GetAccessLogFileWriter(), command()))

	GetManager().GetConfig().configAndServe(GetManager().router)
}

func initCore(config Config) {

	if config.ACCESSLOGFILE == "" {
		config.ACCESSLOGFILE = "access.log"
	}

	//TODO HANDLE TEMPLATE SOURCE DIR
	//router.LoadHTMLGlob("." + string(os.PathSeparator) + config.RESOURCEDIR + "*" + string(os.PathSeparator) + "*")

	//ACCESS LOGGING
	f, err := os.OpenFile(config.ACCESSLOGFILE, os.O_APPEND|os.O_CREATE|os.O_RDWR, os.ModePerm)
	if err != nil {
		log.Fatalln("GO-WOXY Core - Error opening access log file " + config.ACCESSLOGFILE + " : " + err.Error())
	} else {
		GetManager().SetAccessLogFile(f)
	}

	router := mux.NewRouter()
	//router.Use()

	cp := CommandProcessorImpl{}
	cp.Init()
	GetManager().SetCommandProcessor(&cp)
	GetManager().SetRouter(router)
}

//LaunchCore - start core server
func LaunchCore(configPath string) {

	//Load Config
	c := LoadConfigFromPath(configPath)
	c.motd()
	c.generateSecret()

	//Init Go-Woxy core
	initCore(c)

	// SAVE CONFIG
	GetManager().SetConfig(&c)

	// START MODULE SUPERVISOR
	initSupervisor()

	// STEP 4 LOAD MODULES
	go c.loadModules()

	// STEP 5 START SERVER WHERE MODULES WILL REGISTER
	launchServer()
}

func connect() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//READ Body and try to found ConnexionRequest
		var cr com.ConnexionRequest
		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		cr.Decode(buf.Bytes())

		//GET THE MODULE TARGET
		var modC ModuleConfig = GetManager().GetConfig().MODULES[cr.Name]

		var resultW []byte
		if reflect.DeepEqual(modC, ModuleConfig{}) {

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
			rs := hashMatchSecretHash(cr.Secret)
			if rs && cr.ModHash != "" {
				go registerModule(&modC, &cr)
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

		GetManager().SaveModuleChanges(&modC)
	}
}

func error404(w http.ResponseWriter, r *http.Request) {
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

func hashMatchSecretHash(hash string) bool {
	r := strings.Trim(hash, "\n\t") == strings.Trim(GetManager().GetConfig().SECRET, "\n\t")
	return r
}

func registerModule(m *ModuleConfig, cr *com.ConnexionRequest) bool {
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

	if m.RESOURCEPATH == "" {
		m.RESOURCEPATH = "resources/"
	}

	if m.BINDING.PORT == "" || cr.Port != "" {
		m.BINDING.PORT = cr.Port
	}

	//PREPARE PING REQUEST
	cp := GetManager().GetCommandProcessor()
	var crr com.CommandRequest
	crr.Generate("Ping", m.PK, m.NAME, GetManager().GetConfig().SECRET)
	var c interface{} = &crr
	p := (c).(com.Request)

	time.Sleep(time.Second * 10)

	//RETRY 15 TIME TO CHECK MODULE COME ONLINE
	try := 0
	r := false
	for {
		res, e := cp.Run("Ping", &p, m, "")
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
		err = m.HookAll(GetManager().GetRouter())
		if err == nil && m.EXE.SUPERVISED {
			GetManager().AddModuleToSupervisor(m)
		} else if err != nil {
			log.Println("Go-WOXY Core - Error trying to hook module", m.NAME)
		}
	}
	GetManager().SaveModuleChanges(m)
	return r
}

// Command - Access point to handle module commands
func command() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t, b := com.GetCustomRequestType(r)

		from := r.RemoteAddr
		response := ""
		action := ""

		//CHECK AUTH ( TODO API KEY)
		rs := t["Secret"] == GetManager().GetConfig().SECRET

		// CHECK ERROR DURING READING DATA
		if t["error"] == "error" {
			response = "Error reading Request"
		} else if t["Hash"] != "" && rs {

			//GET MOD WITH HASH
			mc := GetManager().SearchModWithHash(t["Hash"])

			if mc.NAME == "error" {
				response = "Error module not found"
			} else {
				action += "To " + mc.NAME + " - "

				//PROCESS REQUEST
				switch t["Type"] {
				case "Command":
					var cr com.CommandRequest
					cr.Decode(b)
					cp := GetManager().GetCommandProcessor()
					var c interface{} = &cr
					p := (c).(com.Request)
					res, e := cp.Run(cr.Command, &p, &mc, "")
					response += res
					if e != nil {
						response = response + " " + e.Error()
					}
					action += "Command [ " + cr.Command + " ]"
				}
				GetManager().SaveModuleChanges(&mc)
			}
		} else {
			if t["Hash"] == "" {
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
