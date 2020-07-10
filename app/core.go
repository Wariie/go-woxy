package app

import (
	"bytes"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	com "guilhem-mateo.fr/go-woxy/app/com"
)

var configFile string

var config Config

var secret string = "SECRET"

//Path -
var Path map[string][]string

//Router -
var Router *gin.Engine

//CORE SOCKET IS THE WHERE ALL THE MODULES EXCHANGE WILL BE TREATED
//ALL THE APP IS CONSTIUED BY MODULES
//THE CORE IS THIS HERE TO HANDLE AND LOG THESE DIFFERENTS MODULES
func launchServer(Router *gin.Engine) {
	log.Print("START SERVER")
	//AUTHENTICATION ENDPOINT
	Router.POST("/connect", connect)

	server := &http.Server{
		Addr:    config.SERVER.ADDRESS + ":" + config.SERVER.PORT + config.SERVER.PATH,
		Handler: Router,
	}
	log.Fatal("ERROR SERVING : ", server.ListenAndServe())
}

func initCore() {

}

//LaunchCore - start core server
func LaunchCore(configPath string) {
	log.Print("STARTING Core")

	// STEP 1 INIT CORE
	initCore()

	// STEP 2 READ CONFIG FILE
	config = readConfig(configPath)

	Router := gin.Default()

	// STEP 4 LOAD MODULES
	go loadModules(Router)

	// STEP 5 START SERVER WHERE MODULES WILL REGISTER
	launchServer(Router)
}

func loadModules(Router *gin.Engine) {
	log.Print("LOAD MODULES")
	for k := range config.MODULES {
		mod := config.MODULES[k]
		log.Print(mod.NAME + " LOADING")
		mod.BuildAndStart()
		mod.Hook(Router)
		config.MODULES[k] = mod
	}
}

func connect(context *gin.Context) {

	var cr com.ConnexionRequest
	buf := new(bytes.Buffer)
	buf.ReadFrom(context.Request.Body)
	cr.Decode(buf.Bytes())
	log.Println(buf.String())
	modC := config.MODULES[cr.Name]

	if modC == (ModuleConfig{}) {
		errMsg := "ERROR Reading ConnexionRequest"
		log.Println(errMsg)
		context.Writer.Write([]byte(errMsg))
	} else {

		modC.SERVER.ADDRESS = strings.Split(context.Request.RemoteAddr, ":")[0]

		tS := cr.Secret == secret
		//CHECK SECRET FOR AUTH
		if tS && cr.ModHash != "" {

			//UPDATE MOD ATTRIBUTES
			modC.pk = cr.ModHash
			modC.STATE = "ONLINE"

			if modC.SERVER.PORT != "" {
				cr.Port = modC.SERVER.PORT
			} else {
				modC.SERVER.PORT = cr.Port
			}

		} else {
			modC.STATE = "FAILED"
			log.Println("")
		}

		//SEND RESPONSE
		var crr com.ConnexionReponseRequest

		result := strconv.FormatBool(tS)
		log.Println("MOD ", modC.NAME, "CONNECT -", result)

		crr.Generate(cr.Name, result, cr.ModHash, cr.Port)
		context.Writer.Write(crr.Encode())
	}

	config.MODULES[cr.Name] = modC
}
