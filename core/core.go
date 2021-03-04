package core

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"path"
	"reflect"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/Wariie/go-woxy/com"
	"github.com/gorilla/mux"
)

func launchServer() {
	fmt.Println("GO-WOXY Core - Starting")

	//AUTHENTICATION ENDPOINT
	GetManager().GetRouter().HandleFunc("/connect", connect)

	//COMMAND ENDPOINT
	GetManager().GetRouter().HandleFunc("/cmd", command)

	log.Fatalln("GO-WOXY Core - Error serving :", GetManager().GetConfig().configAndServe(GetManager().router))
}

func initLogs() {
	//zerolog.SetGlobalLevel(zerolog.InfoLevel)

	//zLog.Logger = zLog.Output(
	//	zerolog.ConsoleWriter{
	//		Out:     os.Stdout,
	//		NoColor: false,
	//	},
	//)
}

func initCore(config Config) {
	//PRODUCTION MODE
	//gin.SetMode(gin.DebugMode)

	//TODO CHANGE LOG AGENT ?
	//router.Use(logger.SetLogger(), gin.Recovery())

	//TODO HANDLE TEMPLATE SOURCE DIR
	//router.LoadHTMLGlob("." + string(os.PathSeparator) + config.RESOURCEDIR + "*" + string(os.PathSeparator) + "*")

	router := mux.NewRouter()
	router.NotFoundHandler = http.HandlerFunc(error404)

	cp := CommandProcessorImpl{}
	cp.Init()
	GetManager().SetCommandProcessor(&cp)
	GetManager().SetRouter(router)
}

//LaunchCore - start core server
func LaunchCore(configPath string) {
	//Init logs
	initLogs()

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

func connect(w http.ResponseWriter, r *http.Request) {

	//READ Body and try to found ConnexionRequest
	var cr com.ConnexionRequest
	buf := new(bytes.Buffer)
	buf.ReadFrom(r.Body)
	cr.Decode(buf.Bytes())

	//GET THE MODULE TARGET
	var modC ModuleConfig
	modC = GetManager().GetConfig().MODULES[cr.Name]

	var resultW []byte
	if reflect.DeepEqual(modC, ModuleConfig{}) {

		//ERROR DURING THE BODY PARSING
		errMsg := "Error reading ConnexionRequest"
		log.Println(errMsg)
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
		fmt.Println("GO-WOXY Core - Module", modC.NAME, "connecting - result :", result)

		cr.State = result
		resultW = cr.Encode()
	}
	i, err := w.Write(resultW)
	if err != nil {
		fmt.Println("GO-WOXY Core - Module", modC.NAME, " failed to respond :", err.Error(), " bytes : ", i)
	}

	GetManager().SaveModuleChanges(&modC)
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

func checkModuleRequestAuth(cr com.ConnexionRequest) bool {
	rs := hashMatchSecretHash(cr.Secret)
	if rs && cr.ModHash != "" {
		return true
	}
	return false
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
	var c interface{}
	c = &crr
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
func command(w http.ResponseWriter, r *http.Request) {
	log.Print("GO-WOXY Core - Command")
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
				var c interface{}
				c = &cr
				p := (c).(com.Request)
				res, e := cp.Run(cr.Command, &p, &mc, "")
				response += res
				if e != nil {
					response += e.Error()
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
	log.Println("From", from, ':', action)

	w.WriteHeader(200)
	w.Write([]byte(response))
}
