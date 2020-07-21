package modbase

import (
	"bytes"
	"log"
	"net/http"
	"strconv"

	gintemplate "github.com/foolin/gin-template"
	"github.com/gin-gonic/gin"

	com "github.com/Wariie/go-woxy/com"
)

//HubAddress - Ip Address of thes hub
var HubAddress = "127.0.0.1"

//HubPort - Communication Port of the hub
var HubPort = "2000"

//ModuleAddress -
var ModuleAddress = "127.0.0.1"

//ModulePort -
var ModulePort = "2501"

type (
	/*HardwareUsage - Module hardware usage */
	HardwareUsage struct {
		CPU     byte
		MEM     byte
		NETWORK int
	}

	/*Module - Module*/
	Module interface {
		GetInfo() ModuleInfo
		GetInstanceName() string
		GetName() string
		Init()
		Register(string, func(*gin.Context), string)
		Run()
		Stop()
	}

	/*ModuleInfo - Module informations*/
	ModuleInfo struct {
		srv com.Server
		fmp string
	}

	/*ModuleImpl - Impl of Module*/
	ModuleImpl struct {
		Name         string
		InstanceName string
		Router       *gin.Engine
		Hash         string
	}
)

//Stop - stop module
func (mod *ModuleImpl) Stop() {
	GetModManager().Shutdown()
}

//Run - start module function
//default ip 	-> 0.0.0.0
//default port	-> 2500
func (mod *ModuleImpl) Run() {
	log.Println("RUN - ", mod.GetName())
	//TODO ADD CONFIG FOR IP AND PORT
	if mod.connectToHub() {
		mod.serve(ModuleAddress, ModulePort)
	} else {
		mod.serve(ModuleAddress, ModulePort)
	}
}

//Init - init module
func (mod *ModuleImpl) Init() {
	GetModManager().SetMod(mod)
	mod.Router = gin.New()
	mod.Router.Use(gin.Logger())
	mod.Router.Use(gin.Recovery())

}

//Register - register http handler for path
func (mod *ModuleImpl) Register(method string, path string, handler gin.HandlerFunc, typeM string) {
	log.Println("REGISTER - ", path)
	mod.Router.Handle(method, path, handler)

	if typeM == "WEB" {
		mod.Router.HTMLRender = gintemplate.Default()
		mod.Router.Static(path+"/ressources/", "./ressources/")
		mod.Router.LoadHTMLGlob("./ressources/html/*.html")
	}
}

//GetName - get module name
func (mod *ModuleImpl) GetName() string {
	return mod.Name
}

//GetInstanceName - get module name
func (mod *ModuleImpl) GetInstanceName() string {
	return mod.InstanceName
}

/*serve -  */
func (mod *ModuleImpl) serve(ip string, port string) {
	mod.Router.POST("/cmd", cmd)
	mod.Router.Run(ip + ":" + port)

	Server := &http.Server{
		Addr:    ip + ":" + port,
		Handler: mod.Router,
	}

	GetModManager().server = Server
	log.Fatal(Server.ListenAndServe())
}

func cmd(c *gin.Context) {
	log.Println("Command request")
	t, b := com.GetCustomRequestType(c.Request)

	var response string

	if t["Hash"] == GetModManager().GetMod().Hash {
		response = "Error reading module Hash"
	} else {

		if t["Type"] == "Shutdown" {
			log.Println("Shutdown")
			var sr com.ShutdownRequest
			sr.Decode(b)
			log.Println("Request Content - ", sr)

			response = "SHUTTING DOWN " + GetModManager().GetMod().InstanceName

			go GetModManager().Shutdown()
		}
	}
	c.String(200, response)
}

func (mod *ModuleImpl) connectToHub() bool {
	log.Println("	HUB CONNECT")

	//CREATE CONNEXION REQUEST
	cr := com.ConnexionRequest{}
	cr.Generate(mod.GetName(), "SECRET", ModulePort)

	//SEND REQUEST
	body := com.SendRequest(com.Server{IP: HubAddress, Port: HubPort, Path: "", Protocol: "http"}, &cr, false)

	var crr com.ConnexionReponseRequest
	crr.Decode(bytes.NewBufferString(body).Bytes())

	s, err := strconv.ParseBool(crr.State)

	if s && err == nil {
		log.Println("		SUCCESS")
		//SET HASH
	} else {
		log.Println("		ERROR - ", err)
	}

	ModulePort = crr.Port
	return s && err == nil
}
