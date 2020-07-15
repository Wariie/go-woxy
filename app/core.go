package app

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	com "guilhem-mateo.fr/git/Wariie/go-woxy.git/app/com"
)

var configFile string

var config Config

var secret string = "SECRET"

//CORE SOCKET IS THE WHERE ALL THE MODULES EXCHANGE WILL BE TREATED
//ALL THE APP IS CONSTIUED BY MODULES
//THE CORE IS THIS HERE TO HANDLE AND LOG THESE DIFFERENTS MODULES
func launchServer(router *gin.Engine) {
	log.Print("START SERVER")
	//AUTHENTICATION ENDPOINT
	router.POST("/connect", connect)

	server := &http.Server{
		Addr:    config.SERVER.ADDRESS + ":" + config.SERVER.PORT + config.SERVER.PATH,
		Handler: router,
	}
	log.Fatal("ERROR SERVING : ", server.ListenAndServe())
}

func motd() {
	fmt.Println(" -------------------- Go-Woxy - V 0.0.1 -------------------- ")
	file, err := os.Open("motd.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}
	fmt.Println("------------------------------------------------------------ ")
}

func initCore() {
	wd, err := os.Getwd()

	//WORKING DIR + "/mods" (NEED ALREADY CREATED DIR (DO AT STARTUP ?))
	os.Mkdir(wd+"/mods", os.ModeDir)
	if err == nil {
		//TODO
	}
}

//LaunchCore - start core server
func LaunchCore(configPath string) {
	motd()

	// STEP 1 Init
	initCore()

	// STEP 2 READ CONFIG FILE
	config = readConfig(configPath)

	router := gin.Default()

	// STEP 4 LOAD MODULES
	go loadModules(router)

	// STEP 5 START SERVER WHERE MODULES WILL REGISTER
	launchServer(router)
}

func loadModules(router *gin.Engine) {
	for k := range config.MODULES {
		mod := config.MODULES[k]
		err := mod.Setup(router)
		if err != nil {
			log.Println(err)
		}
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
