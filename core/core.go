package core

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"

	com "github.com/Wariie/go-woxy/com"
	"github.com/gin-gonic/gin"
)

var configFile string

var motdFileName string = "motd.txt"

var secret string = "SECRET"

//CORE SOCKET IS THE WHERE ALL THE MODULES EXCHANGE WILL BE TREATED
//ALL THE APP IS CONSTIUED BY MODULES
//THE CORE IS THIS HERE TO HANDLE AND LOG THESE DIFFERENTS MODULES
func launchServer() {
	fmt.Print("Start Go-Woxy Server")

	//AUTHENTICATION ENDPOINT
	GetManager().router.POST("/connect", connect)
	GetManager().router.POST("/cmd", command)

	server := getServerConfig(GetManager().config.SERVER, GetManager().router)

	//GetManager().server = server

	log.Fatalln("Error ListenAndServer : ", server.ListenAndServe())
}

func motd() {
	fmt.Println(" -------------------- Go-Woxy - V 0.0.1 -------------------- ")
	file, err := os.Open(motdFileName)
	if err != nil {
		log.Fatalln("No motd file ", motdFileName, " : ", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}
	fmt.Println("------------------------------------------------------------ ")
}

func initCore() {
	//INIT MODULE DIRECTORY
	wd, err := os.Getwd()

	os.Mkdir(wd+"/mods", os.ModeDir)
	if err != nil {
		log.Fatalln("Error creating mods folder : ", err)
		os.Exit(1)
	}

	//INIT ROUTER
	router := gin.Default()
	router.LoadHTMLGlob("ressources/*/*")
	router.NoRoute(func(c *gin.Context) {
		c.HTML(404, "404.html", nil)
	})

	GetManager().SetRouter(router)
}

//LaunchCore - start core server
func LaunchCore(configPath string) {
	motd()

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

func loadModules() {
	config := GetManager().config
	Router := GetManager().router
	for k := range config.MODULES {
		mod := config.MODULES[k]
		err := mod.Setup(Router)
		if err != nil {
			log.Fatalln("Error setup module ", mod.NAME, " - ", err)
		}
		config.MODULES[k] = mod
	}
	GetManager().router = Router
	GetManager().config = config
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

		tS := cr.Secret == secret
		//CHECK SECRET FOR AUTH
		if tS && cr.ModHash != "" {

			//UPDATE MOD ATTRIBUTES
			modC.pk = cr.ModHash
			modC.STATE = "ONLINE"
			log.Println("HASH : ", modC.pk, " - MOD : ", modC.NAME)

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

		result := strconv.FormatBool(tS)
		fmt.Println("Module ", modC.NAME, " connecting - result : ", result)

		crr.Generate(cr.Name, result, cr.ModHash, cr.Port)
		context.Writer.Write(crr.Encode())
	}

	GetManager().config.MODULES[cr.Name] = modC
}

// Command - Access point to manage go-woxy modules
func command(c *gin.Context) {
	log.Print("Go-Woxy Module Command request : ")
	t, b := com.GetCustomRequestType(c.Request)

	log.Println(string(t["Type"]))
	//TODO HANDLE HUB ACCESS WITH CREDENTIALS
	response := ""

	if t["Hash"] == "hub" {
		commandForHub(t, b)
	} else {
		forward := false
		mc := SearchModWithHash(t["Hash"])

		if mc.NAME == "error" {
			response = "Error reading module Hash"
		} else {

			var r com.Request

			switch t["Type"] {
			case "Command":
				var cr com.CommandRequest
				cr.Decode(b)

				switch cr.Command {
				case "Shutdown":
					forward = true
					r = &cr
				case "Log":
					response = mc.GetLog()
				}
			}

			if forward {
				response = com.SendRequest(mc.GetServer(""), r, false)
			}
		}
	}
	c.String(200, response, nil)
}

//TODO
func commandForHub(t map[string]string, b []byte) {

	switch t["Type"] {
	case "Shutdown":
	case "Log":
	case "test":
	}
}

//SearchModWithHash -
func SearchModWithHash(hash string) ModuleConfig {
	mods := GetManager().config.MODULES
	for i := range mods {
		if mods[i].pk == hash {
			return mods[i]
		}
	}
	return ModuleConfig{NAME: "error"}
}
