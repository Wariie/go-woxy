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
	"github.com/gin-gonic/gin"
	"github.com/shiena/ansicolor"
	logrus "github.com/sirupsen/logrus"
	ginlogrus "github.com/toorop/gin-logrus"
)

var configFile string

var motdFileName string = "motd.txt"

var secretHash string = "SECRET"

//CORE SOCKET IS THE WHERE ALL THE MODULES EXCHANGE WILL BE TREATED
//ALL THE APP IS CONSTIUED BY MODULES
//THE CORE IS THIS HERE TO HANDLE AND LOG THESE DIFFERENTS MODULES
func launchServer() {
	fmt.Print("Start Go-Woxy Server")

	//AUTHENTICATION ENDPOINT
	GetManager().router.POST("/connect", connect)
	GetManager().router.POST("/cmd", command)

	server := getServerConfig(GetManager().config.SERVER, GetManager().router)

	log.Fatalln("Error ListenAndServer : ", server.ListenAndServe())
}

func initCore() {

	l := logrus.New()
	//INIT ROUTER
	l.SetFormatter(&logrus.TextFormatter{ForceColors: true})
	l.SetOutput(ansicolor.NewAnsiColorWriter(os.Stdout))

	router := gin.New()
	router.Use(ginlogrus.Logger(l), gin.Recovery())

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

	motd()

	generateSecret()

	// STEP 1 Init
	initCore()

	// STEP 2 READ CONFIG FILE
	config := readConfig(configPath)

	// SAVE CONFIG
	GetManager().config = &config

	// START MODULE SUPERVISOR
	initSupervisor()

	// STEP 4 LOAD MODULES
	go loadModules()

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
		errMsg := "Error reading ConnexionRequest"
		log.Println(errMsg)
		context.Writer.Write([]byte(errMsg))
	} else {

		modC.BINDING.ADDRESS = strings.Split(context.Request.Host, ":")[0]
		rs := strings.TrimSuffix(cr.Secret, "\n\t") == strings.TrimSuffix(secretHash, "\n\t")
		//CHECK SECRET FOR AUTH
		if rs && cr.ModHash != "" {
			go checkModuleOnline(&modC, cr)
		} else {
			modC.STATE = Failed
		}

		//SEND RESPONSE
		var crr com.ConnexionReponseRequest

		result := strconv.FormatBool(rs)
		fmt.Println("Module ", modC.NAME, " connecting - result : ", result)

		crr.Generate(cr.ModHash, cr.Name, cr.Port, result)
		context.Writer.Write(crr.Encode())
	}

	GetManager().SaveModuleChanges(&modC)
}

func checkModuleOnline(m *ModuleConfig, cr com.ConnexionRequest) bool {
	tm := m

	//UPDATE MOD ATTRIBUTES
	pid, err := strconv.Atoi(cr.Pid)
	if err != nil {
		log.Println("Error reading PID :", err)
	}

	m.pid = pid
	m.PK = cr.ModHash
	m.customCommands = cr.CustomCommands
	m.STATE = "ONLINE"
	log.Println("HASH :", m.PK, "- MOD :", m.NAME)

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
	crr.Generate("Ping", m.PK, m.NAME, secretHash)
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

// Command - Access point to manage go-woxy modules
func command(c *gin.Context) {
	log.Print("Go-Woxy Module Command request : ")
	t, b := com.GetCustomRequestType(c.Request)

	from := c.Request.RemoteAddr

	//TODO HANDLE ACCESS WITH CREDENTIALS
	response := ""
	action := ""

	rs := strings.TrimSuffix(t["Secret"], "\n\t ") == strings.TrimSuffix(secretHash, "\n\t ")

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
	log.Println("Request from", from, "-", action)
	c.String(200, "%s", response)
}
