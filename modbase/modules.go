package modbase

import (
	"bytes"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	gintemplate "github.com/foolin/gin-template"
	"github.com/gin-gonic/gin"

	com "github.com/Wariie/go-woxy/app/com"
)

//HubAddress - Ip Address of thes hub
var HubAddress = "127.0.0.1"

//HubPort - Communication Port of the hub
var HubPort = "2000"

//ModuleAddress -
var ModuleAddress = "127.0.0.1"

//ModulePort -
var ModulePort = "2501"

//ModT -
var ModT *ModuleImpl

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
	}
)

//Stop - stop module
func (mod *ModuleImpl) Stop() {

	//WAIT 2 SECOND FOR LAST HEADER REPONSE TO BE SENT
	time.Sleep(2 * time.Second)

	//KILL MODULE
	os.Exit(0)
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
	ModT = mod
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
		mod.Router.Static(path+"/css/", "./ressources/css")
		mod.Router.Static(path+"/img/", "./ressources/img")
		mod.Router.Static(path+"/js/", "./ressources/js")
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
	mod.Router.POST("/ap", accessPoint)
	mod.Router.POST("/shutdown", shutdown)
	mod.Router.Run(ip + ":" + port)

	Server := &http.Server{
		Addr:    ip + ":" + port,
		Handler: mod.Router,
	}
	log.Fatal(Server.ListenAndServe())
}

func accessPoint(c *gin.Context) {
	log.Println("ACCESS POINT CALL")
}

func shutdown(c *gin.Context) {
	log.Println("SHUTTING DOWN - FROM ", c.Request.RemoteAddr)

	var sr com.ShutdownRequest
	buf := new(bytes.Buffer)
	buf.ReadFrom(c.Request.Body)
	sr.Decode(buf.Bytes())
	log.Println(sr)

	var b []byte
	b = bytes.NewBufferString("SHUTTING DOWN " + ModT.InstanceName).Bytes()

	c.Writer.Write(b)
	c.AbortWithStatus(205)

	go ModT.Stop()
}

func (mod *ModuleImpl) connectToHub() bool {
	log.Println("	HUB CONNECT")

	//CREATE CONNEXION REQUEST
	cr := com.ConnexionRequest{}
	cr.Generate(mod.GetName(), "SECRET", ModulePort)

	//SEND REQUEST
	body := com.SendRequest(com.Server{IP: HubAddress, Port: HubPort, Path: "", Protocol: "http"}, &cr)

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
