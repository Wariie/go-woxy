package core

import (
	"bytes"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"

	"github.com/Wariie/go-woxy/com"
	"github.com/gin-gonic/gin"
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
	//INIT ROUTER
	router := gin.Default()
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

	//SAVE CONFIG
	man := GetManager()
	man.config = config

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
		s := secretHash
		rs := strings.TrimSuffix(cr.Secret, "\n\t") == strings.TrimSuffix(s, "\n\t")
		//CHECK SECRET FOR AUTH
		if rs && cr.ModHash != "" {

			//UPDATE MOD ATTRIBUTES
			pid, err := strconv.Atoi(cr.Pid)
			if err != nil {
				log.Println("Error reading PID :", err)
			}
			modC.pid = pid
			modC.PK = cr.ModHash
			modC.STATE = "ONLINE"
			log.Println("HASH :", modC.PK, "- MOD :", modC.NAME)

			if modC.BINDING.PORT != "" {
				cr.Port = modC.BINDING.PORT
			} else {
				modC.BINDING.PORT = cr.Port
			}

		} else {
			modC.STATE = "FAILED"
			log.Println("")
		}

		//SEND RESPONSE
		var crr com.ConnexionReponseRequest

		result := strconv.FormatBool(rs)
		fmt.Println("Module ", modC.NAME, " connecting - result : ", result)

		crr.Generate(cr.ModHash, cr.Name, cr.Port, result)
		context.Writer.Write(crr.Encode())
	}

	GetManager().config.MODULES[cr.Name] = modC
}

// Command - Access point to manage go-woxy modules
func command(c *gin.Context) {
	log.Print("Go-Woxy Module Command request : ")
	t, b := com.GetCustomRequestType(c.Request)

	from := c.Request.RemoteAddr
	//TODO HANDLE ACCESS WITH CREDENTIALS
	response := ""
	action := ""

	// IF ERROR READING DATA
	if t["error"] == "error" {
		response = "Error reading module Hash"
	} else if t["Hash"] != "" {
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
				res, e := cp.Run(cr.Command, &cr, &mc, "")
				response += res
				if e != nil {
					response += e.Error()
				}
				action += "Command [ " + cr.Command + " ]"
			}
		}
		//NO HASH PROVIDED
	} else {
		response = "Empty Hash : Try to start module"
	}
	action += " - Result : " + response
	log.Println("Request from", from, "-", action)
	c.String(200, response, nil)
}
