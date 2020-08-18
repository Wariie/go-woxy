package core

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	com "github.com/Wariie/go-woxy/com"
	"github.com/Wariie/go-woxy/tools"
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

func generateSecret() {
	s := []byte(tools.String(64))
	err := ioutil.WriteFile(".secret", s, 0644)
	if err != nil {
		log.Println("Error trying create secret file :", err)
	}
	b := sha256.Sum256(s)
	secretHash = string(b[:])
}

func loadModules() {
	config := GetManager().config
	Router := GetManager().router
	for k := range config.MODULES {
		mod := config.MODULES[k]
		err := mod.Setup(Router, true)
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
		rs := bytes.Compare([]byte(strings.Trim(cr.Secret, " ")), []byte(secretHash))
		//CHECK SECRET FOR AUTH
		tS := math.Abs(float64(rs)) == 1
		if tS && cr.ModHash != "" {

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

	from := c.Request.RemoteAddr
	//TODO HANDLE HUB ACCESS WITH CREDENTIALS
	response := ""
	action := ""

	if t["Hash"] == "hub" {
		response = commandForHub(t, b)
	} else if t["Hash"] != "" {
		forward := false

		if t["error"] == "error" {
			response = "Error reading module Hash"
		} else {
			mc := SearchModWithHash(t["Hash"])

			var r com.Request
			if mc.NAME == "error" {
				response = "Error module not found"
			} else {
				action += "To " + mc.NAME + " - "

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
					case "Restart":
						cr.Command = "Shutdown"
						r = &cr
						rqtS, err := com.SendRequest(mc.GetServer("/cmd"), r, false)
						mc.STATE = Stopped
						if strings.Contains(rqtS, "SHUTTING DOWN "+mc.NAME) || (err != nil && strings.Contains(err.Error(), "An existing connection was forcibly closed by the remote host")) {
							time.Sleep(10 * time.Second)
							if err := mc.Setup(GetManager().GetRouter(), false); err != nil {
								response += "Error :" + err.Error()
								log.Panicln(err)
							} else {
								response += "Success"
							}
						} else {
							response += "Error :" + rqtS
							if err != nil {
								response += " - " + err.Error()
							}
						}
					case "Performance":
						c, r := mc.GetPerf()
						response += "CPU/RAM : " + fmt.Sprintf("%f", c) + "/" + fmt.Sprintf("%f", r)
					}

					action += "Command [ " + cr.Command + " ]"
				}
			}

			if forward {
				resp, err := com.SendRequest(mc.GetServer("/cmd"), r, false)
				response += resp
				if err != nil {
					response += err.Error()
				}
			}

		}
	} else {
		response = "Empty Hash : Try to start module"
	}
	action += " - Result : " + response
	log.Println("Request from", from, "-", action)
	c.String(200, response, nil)
}

//TODO
func commandForHub(t map[string]string, b []byte) string {

	switch t["Type"] {
	case "Command":
		var cr com.CommandRequest
		cr.Decode(b)

		if strings.Contains(cr.Command, "Get") {
			todo := strings.Split(cr.Command, ":")
			if len(todo) == 3 {
				switch todo[1] {
				case "List":
					switch todo[2] {
					case "Module":
						rb, err := json.Marshal(GetManager().GetConfig().MODULES)
						if err != nil {
							return "Error JSON - 420"
						}
						return string(rb)
					}
				}
			}
		}
	}

	return ""
}

//SearchModWithHash -
func SearchModWithHash(hash string) ModuleConfig {
	mods := GetManager().config.MODULES
	for i := range mods {
		if mods[i].PK == hash {
			return mods[i]
		}
	}
	return ModuleConfig{NAME: "error"}
}
