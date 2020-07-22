package modbase

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	gintemplate "github.com/foolin/gin-template"
	"github.com/gin-contrib/static"
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
func (mod *ModuleImpl) Stop(c *gin.Context) {
	GetModManager().Shutdown(c)
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
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	GetModManager().SetRouter(r)
}

//Register - register http handler for path
func (mod *ModuleImpl) Register(method string, path string, handler gin.HandlerFunc, typeM string) {
	log.Println("REGISTER - ", path)
	r := GetModManager().GetRouter()
	r.Handle(method, path, handler)

	if typeM == "WEB" {
		r.HTMLRender = gintemplate.Default()
		r.Use(static.ServeRoot(path+"ressources/", "./ressources/"))
		//mod.Router.Static(path+"/ressources/", "./ressources/")
		r.LoadHTMLGlob("./ressources/html/*.html")
	}
	GetModManager().SetRouter(r)
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
	r := GetModManager().GetRouter()
	r.POST("/cmd", cmd)

	Server := &http.Server{
		Addr:    ip + ":" + port,
		Handler: r,
	}

	GetModManager().SetServer(Server)
	GetModManager().SetRouter(r)

	if err := GetModManager().GetServer().ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal(err)
	}

}

func cmd(c *gin.Context) {
	log.Println("Command request")
	t, b := com.GetCustomRequestType(c.Request)

	var response string

	if t["Hash"] == GetModManager().GetMod().Hash {
		response = "Error reading module Hash"
	} else {

		switch t["Type"] {
		case "Command":
			var sr com.CommandRequest
			sr.Decode(b)
			log.Println("Request Content - ", sr)
			switch sr.Command {
			case "Shutdown":
				response = "SHUTTING DOWN " + GetModManager().GetMod().Name
				go GetModManager().Shutdown(c)
			}
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
	body, err := com.SendRequest(com.Server{IP: HubAddress, Port: HubPort, Path: "", Protocol: "http"}, &cr, false)

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

type modManager struct {
	server *http.Server
	router *gin.Engine
	mod    *ModuleImpl
}

var singleton *modManager
var once sync.Once

//GetModManager -
func GetModManager() *modManager {
	once.Do(func() {
		singleton = &modManager{}
	})
	return singleton
}

func (sm *modManager) GetServer() *http.Server {
	return sm.server
}

func (sm *modManager) SetServer(s *http.Server) {
	sm.server = s
}

func (sm *modManager) GetRouter() *gin.Engine {
	return sm.router
}

func (sm *modManager) SetRouter(r *gin.Engine) {
	sm.router = r
}

func (sm *modManager) SetMod(m *ModuleImpl) {
	sm.mod = m
}

func (sm *modManager) GetMod() *ModuleImpl {
	return sm.mod
}

func (sm *modManager) Shutdown(c context.Context) {
	time.Sleep(10 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := sm.server.Shutdown(ctx); err != nil {
		log.Fatal("Server force to shutdown:", err)
	}
	log.Println("Server exiting")
}
