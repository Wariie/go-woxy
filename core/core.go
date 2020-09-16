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
	"github.com/rs/zerolog"
	zLog "github.com/rs/zerolog/log"
)

func launchServer() {
	fmt.Println("GO-WOXY Core - Starting")

	//AUTHENTICATION ENDPOINT
	GetManager().router.POST("/connect", connect)
	GetManager().router.POST("/cmd", command)

	log.Fatalln("GO-WOXY Core - Error ListenAndServer :", GetManager().config.configAndServe(GetManager().router))
}

func initCore() {

	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	zLog.Logger = zLog.Output(
		zerolog.ConsoleWriter{
			Out:     os.Stdout,
			NoColor: false,
		},
	)

	//PRODUCTION MODE
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(logger.SetLogger(), gin.Recovery())
	router.LoadHTMLGlob("ressources/*/*")
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

	// STEP 1 Init
	initCore()

	var c Config

	c.loadConfig(configPath)

	c.motd()

	c.generateSecret()

	// SAVE CONFIG
	GetManager().config = &c

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
	modC = GetManager().config.MODULES[cr.Name]

	if reflect.DeepEqual(modC, ModuleConfig{}) {
		errMsg := "GO-WOXY Core - Error reading ConnexionRequest"
		log.Println(errMsg)
		context.Writer.Write([]byte(errMsg))
	} else {

		modC.BINDING.ADDRESS = strings.Split(context.Request.Host, ":")[0]

		//CHECK SECRET FOR AUTH
		rs := hashMatchSecretHash(cr.Secret)
		if rs && cr.ModHash != "" {
			go registerModule(&modC, &cr)
		} else {
			modC.STATE = Failed
		}

		//SEND RESPONSE
		var crr com.ConnexionReponseRequest

		result := strconv.FormatBool(rs)
		fmt.Println("GO-WOXY Core - Module ", modC.NAME, " connecting - result : ", result)

		crr.Generate(cr.ModHash, cr.Name, cr.Port, result)
		context.Writer.Write(crr.Encode())
	}

	GetManager().SaveModuleChanges(&modC)
}

func hashMatchSecretHash(hash string) bool {
	r := strings.TrimSuffix(hash, "\n\t") == strings.TrimSuffix(GetManager().GetConfig().SECRET, "\n\t")
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

	if m.BINDING.PORT != "" {
		cr.Port = m.BINDING.PORT
	} else {
		m.BINDING.PORT = cr.Port
	}

	if m.EXE.SUPERVISED {
		GetManager().GetSupervisor().Add(m.NAME)
	}

	//PREPARE PING REQUEST
	cp := GetManager().GetCommandProcessor()
	var crr com.CommandRequest
	crr.Generate("Ping", m.PK, m.NAME, GetManager().GetConfig().SECRET)
	var c interface{}
	c = &crr
	p := ((c).(com.Request))

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
	}
	GetManager().SaveModuleChanges(m)

	return r
}

// Command - Access point to handle module commands
func command(c *gin.Context) {
	log.Print("GO-WOXY Core - Command")
	t, b := com.GetCustomRequestType(c.Request)

	from := c.Request.RemoteAddr

	//TODO HANDLE ACCESS WITH CREDENTIALS
	response := ""
	action := ""

	rs := strings.TrimSuffix(t["Secret"], "\n\t ") == strings.TrimSuffix(GetManager().GetConfig().SECRET, "\n\t ")

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
	log.Println(" from", from, ":", action, "-", response)
	c.String(200, "%s", response)
}
