package modbase

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/foolin/goview/supports/ginview"
	"github.com/gin-contrib/logger"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	zLog "github.com/rs/zerolog/log"

	"github.com/Wariie/go-woxy/com"
)

type (

	/*Module - Module*/
	Module interface {
		Init()
		Register(string, func(*gin.Context), string)
		Run()
		Stop()
		SetServer()
		SetHubServer()
		SetCommand(string, func(r *com.Request, c *gin.Context, mod *ModuleImpl) (string, error))
	}

	/*ModuleImpl - Impl of Module*/
	ModuleImpl struct {
		Name           string
		InstanceName   string
		Router         *gin.Engine
		Hash           string
		Secret         string
		HubServer      com.Server
		Server         com.Server
		RessourcePath  string
		CustomCommands map[string]func(r *com.Request, c *gin.Context, mod *ModuleImpl) (string, error)
	}
)

//Stop - stop module
func (mod *ModuleImpl) Stop(c *gin.Context) {
	GetModManager().Shutdown(c)
}

//SetCommand - set command
func (mod *ModuleImpl) SetCommand(name string, run func(r *com.Request, c *gin.Context, mod *ModuleImpl) (string, error)) {
	if mod.CustomCommands == nil {
		mod.CustomCommands = map[string]func(r *com.Request, c *gin.Context, mod *ModuleImpl) (string, error){}
	}
	mod.CustomCommands[name] = run
}

//SetServer -
func (mod *ModuleImpl) SetServer(ip string, path string, port string, proto string) {
	if proto == "" {
		proto = "http"
	}
	mod.Server = com.Server{IP: com.IP(ip), Port: com.Port(port), Path: com.Path(path), Protocol: com.Protocol(proto)}
}

//SetAddress - Set address for server
func (mod *ModuleImpl) SetAddress(addr string) {
	mod.Server.IP = com.IP(addr)
}

//SetPort - Set port for server
func (mod *ModuleImpl) SetPort(port string) {
	mod.Server.Port = com.Port(port)
}

//SetProtocol - Set protocol for server
func (mod *ModuleImpl) SetProtocol(proto string) {
	mod.Server.Protocol = com.Protocol(proto)
}

//SetPath - Set path for server
func (mod *ModuleImpl) SetPath(path string) {
	mod.Server.Path = com.Path(path)
}

//SetHubServer -
func (mod *ModuleImpl) SetHubServer(ip string, path string, port string, proto string) {
	if proto == "" {
		proto = "http"
	}
	mod.HubServer = com.Server{IP: com.IP(ip), Port: com.Port(port), Path: com.Path(path), Protocol: com.Protocol(proto)}
}

//SetHubAddress - Set address for hub server
func (mod *ModuleImpl) SetHubAddress(addr string) {
	mod.HubServer.IP = com.IP(addr)
}

//SetHubPort - Set port for hub server
func (mod *ModuleImpl) SetHubPort(port string) {
	mod.HubServer.Port = com.Port(port)
}

//SetHubProtocol - Set protocol for hub server
func (mod *ModuleImpl) SetHubProtocol(proto string) {
	mod.HubServer.Protocol = com.Protocol(proto)
}

//SetHubPath - Set path for hub server
func (mod *ModuleImpl) SetHubPath(path string) {
	mod.HubServer.Path = com.Path(path)
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

	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	zLog.Logger = zLog.Output(
		zerolog.ConsoleWriter{
			Out:     os.Stdout,
			NoColor: false,
		},
	)

	r := gin.New()
	r.Use(logger.SetLogger(), gin.Recovery())

	GetModManager().SetRouter(r)
	GetModManager().SetMod(mod)

	mod.readSecret()

	if mod.RessourcePath == "" {
		mod.RessourcePath = "ressources/"
	}

	//DEFAULT MODULE SERVER PARAMETER
	if mod.Server == (com.Server{}) {
		mod.Server = com.Server{IP: "0.0.0.0", Port: "4224", Protocol: "http"}
	}

	//DEFAULT HUB SERVER PARAMETERS
	if mod.HubServer == (com.Server{}) {
		mod.HubServer = com.Server{IP: "0.0.0.0", Port: "2000", Protocol: "http"}
	} else if mod.HubServer.Protocol == "https" && mod.HubServer.Port == "" {
		mod.HubServer.Port = "443"
	}
}

func (mod *ModuleImpl) readSecret() {
	b, err := ioutil.ReadFile(".secret")
	if err != nil {
		log.Println("Error reading server secret")
		os.Exit(2)
	}
	h := sha256.New()
	h.Write(b)
	mod.Secret = base64.URLEncoding.EncodeToString(h.Sum(nil))
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
		r.HTMLRender = ginview.Default()
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
		Addr:    string(s.IP) + ":" + string(s.Port),
		Handler: r,
	}

	GetModManager().SetServer(Server)
	GetModManager().SetRouter(r)

	if err := Server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func (mod *ModuleImpl) connectToHub() bool {
	log.Println("HUB CONNECT")

	//CREATE CONNEXION REQUEST
	cr := com.ConnexionRequest{}

	var commands []string
	for k := range mod.CustomCommands {
		commands = append(commands, k)
	}

	cr.Generate(commands, mod.Name, string(mod.Server.Port), strconv.Itoa(os.Getpid()), mod.Secret)
	mod.Hash = cr.ModHash

	//SEND REQUEST
	body, err := com.SendRequest(com.Server{IP: mod.HubServer.IP, Port: mod.HubServer.Port, Path: "", Protocol: mod.HubServer.Protocol}, &cr, false)

	var crr com.ConnexionReponseRequest
	crr.Decode(bytes.NewBufferString(body).Bytes())

	s, err := strconv.ParseBool(crr.State)

	if s && err == nil {
		log.Println("	SUCCESS")
		//SET HASH
	} else {
		log.Println("	ERROR - ", err)
	}

	mod.Server.Port = com.Port(crr.Port)

	GetModManager().SetMod(mod)

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
	time.Sleep(5 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := sm.server.Shutdown(ctx); err != nil {
		log.Fatal("Server force to shutdown:", err)
	}
	log.Println("Server exiting")
}
