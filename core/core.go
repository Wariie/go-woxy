package core

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/Wariie/go-woxy/com"
	"github.com/gin-contrib/logger"
	"github.com/gin-gonic/gin"
)

func launchServer() {
	fmt.Println("GO-WOXY Core - Starting")

	//AUTHENTICATION ENDPOINT
	GetManager().GetRouter().POST("/connect", connect)

	//COMMAND ENDPOINT
	GetManager().GetRouter().POST("/cmd", command)

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
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(logger.SetLogger(), gin.Recovery())
	router.LoadHTMLGlob("." + string(os.PathSeparator) + config.RESOURCEDIR + "*" + string(os.PathSeparator) + "*")
	router.NoRoute(func(c *gin.Context) {
		c.HTML(404, "404.html", nil)
	})

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

func connect(context *gin.Context) {

	var cr com.ConnexionRequest
	buf := new(bytes.Buffer)
	buf.ReadFrom(context.Request.Body)
	cr.Decode(buf.Bytes())

	var modC ModuleConfig

	modC = GetManager().GetConfig().MODULES[cr.Name]

	var resultW []byte
	if reflect.DeepEqual(modC, ModuleConfig{}) {
		errMsg := "Error reading ConnexionRequest"
		log.Println(errMsg)
		resultW = []byte(errMsg)
	} else {

		if !modC.EXE.REMOTE {
			modC.BINDING.ADDRESS = strings.Split(context.Request.Host, ":")[0]
		}

		//CHECK SECRET FOR AUTH
		//TODO SET API KEY MECANISM
		//cr.Secret --> API KEY corresponding

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
	i, err := context.Writer.Write(resultW)
	if err != nil {
		fmt.Println("Go-WOXY Core - Module", modC.NAME, " failed to respond :", err.Error(), " bytes : ", i)
	}

	GetManager().SaveModuleChanges(&modC)
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
func command(c *gin.Context) {
	log.Print("GO-WOXY Core - Command")
	t, b := com.GetCustomRequestType(c.Request)

	from := c.Request.RemoteAddr

	response := ""
	action := ""

	rs := t["Secret"] == GetManager().GetConfig().SECRET

	// IF ERROR READING DATA
	if t["error"] == "error" {
		response = "Error reading Request"
	} else if t["Hash"] != "" && rs {
		//GET MOD WITH HASH
		mc := searchModWithHash(t["Hash"])

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
	log.Println("From", from, ':', action)
	c.String(200, "%s", response)
}
