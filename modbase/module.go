package modbase

import (
	"bytes"
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	gintemplate "github.com/foolin/gin-template"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"

	com "github.com/Wariie/go-woxy/com"
)

type (
	/*HardwareUsage - Module hardware usage */
	HardwareUsage struct {
		CPU     byte
		MEM     byte
		NETWORK int
	}

	/*Module - Module*/
	Module interface {
		Init()
		Register(string, func(*gin.Context), string)
		Run()
		Stop()
		SetServer()
		SetHubServer()
	}

	/*ModuleImpl - Impl of Module*/
	ModuleImpl struct {
		Name          string
		InstanceName  string
		Router        *gin.Engine
		Hash          string
		Secret        string
		HubServer     com.Server
		Server        com.Server
		RessourcePath string
	}
)

//Stop - stop module
func (mod *ModuleImpl) Stop(c *gin.Context) {
	GetModManager().Shutdown(c)
}

//SetServer -
func (mod *ModuleImpl) SetServer(ip string, port string) {
	mod.Server = com.Server{IP: ip, Port: port}
}

//SetHubServer -
func (mod *ModuleImpl) SetHubServer(ip string, port string) {
	mod.HubServer = com.Server{IP: ip, Port: port}
}

//Run - start module function
func (mod *ModuleImpl) Run() {
	log.Println("RUN - ", mod.Name)
	if mod.connectToHub() {
		mod.serve()
	}
}

//Init - init module
func (mod *ModuleImpl) Init() {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	GetModManager().SetRouter(r)
	GetModManager().SetMod(mod)

	mod.readSecret()

	if mod.RessourcePath == "" {
		mod.RessourcePath = "ressources/"
	}

	if mod.Server == com.Server{} {
		mod.Server = com.Server{IP: "0.0.0.0", Port: "4224"}
	}

	if mod.HubServer == com.Server{} {
		mod.HubServer = com.Server{IP: "0.0.0.0", Port: "2000"}
	}
}

func (mod *ModuleImpl) readSecret() {
	b, err := ioutil.ReadFile(".secret")
	if err != nil {
		log.Println("Error reading server secret")
		os.Exit(2)
	}
	/*bs := sha256.Sum256(b)
	mod.Secret = string(bs[:])*/
	mod.Secret = string(b)
}

//Register - register http handler for path
func (mod *ModuleImpl) Register(method string, path string, handler gin.HandlerFunc, typeM string) {
	log.Println("REGISTER - ", path)
	r := GetModManager().GetRouter()
	r.Handle(method, path, handler)

	if typeM == "WEB" {
		if len(path) > 1 {
			path += "/"
		}
		r.HTMLRender = gintemplate.Default()
		r.Use(static.ServeRoot(path+mod.RessourcePath, "./"+mod.RessourcePath))
	}
	GetModManager().SetRouter(r)
}

/*serve -  */
func (mod *ModuleImpl) serve() {
	r := GetModManager().GetRouter()
	s := GetModManager().GetMod().Server
	r.POST("/cmd", cmd)

	Server := &http.Server{
		Addr:    s.IP + ":" + s.Port,
		Handler: r,
	}

	GetModManager().SetServer(Server)
	GetModManager().SetRouter(r)
	GetModManager().SetMod(mod)

	if err := Server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal(err)
	}

}

func (mod *ModuleImpl) connectToHub() bool {
	log.Println("	HUB CONNECT")

	//CREATE CONNEXION REQUEST
	cr := com.ConnexionRequest{}
	cr.Generate(mod.Name, mod.Server.Port, strconv.Itoa(os.Getpid()), mod.Secret)
	mod.Hash = cr.ModHash

	//SEND REQUEST
	body, err := com.SendRequest(com.Server{IP: mod.HubServer.IP, Port: mod.HubServer.Port, Path: "", Protocol: "http"}, &cr, false)

	var crr com.ConnexionReponseRequest
	crr.Decode(bytes.NewBufferString(body).Bytes())

	s, err := strconv.ParseBool(crr.State)

	if s && err == nil {
		log.Println("		SUCCESS")
		//SET HASH
	} else {
		log.Println("		ERROR - ", err)
	}

	mod.Server.Port = crr.Port
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

func (sm *modManager) GetSecret() string {
	return sm.mod.Secret
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
