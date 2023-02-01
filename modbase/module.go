package modbase

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gorilla/mux"

	"github.com/Wariie/go-woxy/com"
)

type (

	/*Module - Module*/
	Module interface {
		Init()
		Register(string, func(w http.ResponseWriter, r *http.Request), string)
		Run()
		Stop()
		SetServer()
		SetHubServer()
		SetCommand(string, func(r *com.Request, w http.ResponseWriter, re *http.Request, mod *ModuleImpl) (string, error))
	}

	/*ModuleImpl - Impl of Module*/
	ModuleImpl struct {
		Mode           string
		Name           string
		InstanceName   string
		Router         *mux.Router
		Hash           string
		Secret         string
		HubServer      com.Server
		Server         com.Server
		ResourcePath   string
		Certs          []string
		CustomCommands map[string]func(r *com.Request, w http.ResponseWriter, re *http.Request, mod *ModuleImpl) (string, error)
	}
)

// Stop - stop module
func (mod *ModuleImpl) Stop(w http.ResponseWriter, r *http.Request) {
	GetModManager().Shutdown(w)
}

// SetCommand - set command
func (mod *ModuleImpl) SetCommand(name string, run func(r *com.Request, w http.ResponseWriter, re *http.Request, mod *ModuleImpl) (string, error)) {
	if mod.CustomCommands == nil {
		mod.CustomCommands = map[string]func(r *com.Request, w http.ResponseWriter, re *http.Request, mod *ModuleImpl) (string, error){}
	}
	mod.CustomCommands[name] = run
}

// SetServer -
func (mod *ModuleImpl) SetServer(ip string, path string, port string, proto string) {
	if proto == "" {
		proto = "http"
	}
	mod.Server = com.Server{IP: com.IP(ip), Port: com.Port(port), Path: com.Path(path), Protocol: com.Protocol(proto)}
}

// SetAddress - Set address for server
func (mod *ModuleImpl) SetAddress(addr string) {
	mod.Server.IP = com.IP(addr)
}

// SetCerts - Set certificate and key for server
func (mod *ModuleImpl) SetCerts(keyPath string, certPath string) {
	mod.Certs = make([]string, 0)
	mod.Certs = append(mod.Certs, certPath)
	mod.Certs = append(mod.Certs, keyPath)
}

// SetPort - Set port for server
func (mod *ModuleImpl) SetPort(port string) {
	mod.Server.Port = com.Port(port)
}

// SetProtocol - Set protocol for server
func (mod *ModuleImpl) SetProtocol(proto string) {
	mod.Server.Protocol = com.Protocol(proto)
}

// SetPath - Set path for server
func (mod *ModuleImpl) SetPath(path string) {
	mod.Server.Path = com.Path(path)
}

// SetHubServer -
func (mod *ModuleImpl) SetHubServer(ip string, path string, port string, proto string) {
	if proto == "" {
		proto = "http"
	}
	mod.HubServer = com.Server{IP: com.IP(ip), Port: com.Port(port), Path: com.Path(path), Protocol: com.Protocol(proto)}
}

// SetHubAddress - Set address for hub server
func (mod *ModuleImpl) SetHubAddress(addr string) {
	mod.HubServer.IP = com.IP(addr)
}

// SetHubPort - Set port for hub server
func (mod *ModuleImpl) SetHubPort(port string) {
	mod.HubServer.Port = com.Port(port)
}

// SetHubProtocol - Set protocol for hub server
func (mod *ModuleImpl) SetHubProtocol(proto string) {
	mod.HubServer.Protocol = com.Protocol(proto)
}

// SetHubPath - Set path for hub server
func (mod *ModuleImpl) SetHubPath(path string) {
	mod.HubServer.Path = com.Path(path)
}

// Run - start module function
func (mod *ModuleImpl) Run() {
	log.Println("RUN - ", mod.Name)
	if mod.Mode == "Test" || mod.connectToHub() {
		mod.serve()
	}
}

// Init - init module
func (mod *ModuleImpl) Init() {

	r := NewRouter(notFound())
	//r.StrictSlash(true)

	//TODO SET LOGGER
	//r.Use(logger.SetLogger(), gin.Recovery())

	GetModManager().SetRouter(r)
	GetModManager().SetMod(mod)

	mod.readSecret()

	if mod.ResourcePath == "" {
		mod.ResourcePath = "/resources"
	}

	//DEFAULT MODULE SERVER PARAMETER
	if mod.Server == (com.Server{}) {
		mod.Server = com.Server{IP: "0.0.0.0", Port: "4224", Protocol: "http"}
	} else if len(mod.Certs) == 2 {
		if mod.Server.Port == "" {
			mod.Server.Port = "443"
		}
		mod.Server.Protocol = "https"
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

type HttpServer struct {
	http.Server
	shutdownReq chan bool
	reqCount    uint32
}

// TODO ADD CUSTOM LOGGING
// Register - register http handler for path
func (mod *ModuleImpl) Register(path string, handler HandlerFunc, typeM string) {
	log.Println("REGISTER - ", path)
	r := GetModManager().GetRouter()

	if typeM == "WEB" {
		if len(path) == 1 {
			path = ""
		}

		//TODO CHECK IF DISABLE SERVER RESOURCES

		r.Handle(path, resources(path, mod.ResourcePath), &Route{TO: path})
	}

	r.Handle(path, handler, &Route{TO: path})
}

/*serve -  */
func (mod *ModuleImpl) serve() {

	GetModManager().SortRoutes()
	r := GetModManager().GetRouter()
	s := GetModManager().GetMod().Server
	r.Handle("/cmd", cmd(), &Route{TO: "/cmd"})

	server := &HttpServer{
		Server: http.Server{
			Addr:         string(s.IP) + ":" + string(s.Port),
			Handler:      r,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
		shutdownReq: make(chan bool),
	}

	//IF TEST MODE DON'T LAUNCH HUB CHECKING
	if mod.Mode != "Test" {
		go checkHubRunning(mod.HubServer, mod)
	}

	var listener net.Listener

	if len(mod.Certs) == 2 { //CERTIFCATE AND KEY DETECTED
		var cfg tls.Config
		cer, err := tls.LoadX509KeyPair(mod.Certs[0], mod.Certs[1])
		if err != nil {
			log.Println("Error creating tls config :", err)
			cfg = tls.Config{}
		} else {
			cfg = tls.Config{Certificates: []tls.Certificate{cer}}
		}

		//server.TLSConfig = &cfg

		listener, err = tls.Listen("tcp", string(s.IP)+":"+string(s.Port), &cfg)
		if err != nil {
			log.Fatalln("Error setuping https listener :", err)
		}
	} else {

		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalln("Error setupping http listener :", err)
		}
	}

	server.Serve(listener)

	GetModManager().SetServer(server)

	done := make(chan bool)
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			log.Printf("Listen and serve: %v", err)
		}
		done <- true
	}()

	//wait shutdown
	server.WaitShutdown()

	<-done
}

func checkHubRunning(hubServer com.Server, mod *ModuleImpl) {
	retry := 0

	for {
		cr := com.CommandRequest{}
		cr.Generate("Ping", "hub", mod.Name, mod.Secret)
		body, err := com.SendRequest(hubServer, &cr, false)
		if !strings.Contains(body, "Pong") || err != nil {
			if retry > 15 {
				log.Fatalf("Hub not responding after " + strconv.Itoa(retry) + " retries")
			}
			time.Sleep(time.Second)
			log.Println("Cannot access hub : not responding ")
			retry++
		} else {
			retry = 0
		}

		time.Sleep(time.Minute)
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

	if err != nil {
		log.Println("	ERROR - ", err)
	} else {
		var crr com.ConnexionRequest
		crr.Decode(bytes.NewBufferString(body).Bytes())
		s, err := strconv.ParseBool(crr.State)

		if s && err == nil {
			log.Println("	SUCCESS")
			GetModManager().SetMod(mod)
		} else {
			log.Println("	ERROR - ", err)
		}
		return s && err == nil
	}
	return false
}

type modManager struct {
	server *HttpServer
	router *Router
	mod    *ModuleImpl
}

var singleton *modManager
var once sync.Once

// GetModManager -
func GetModManager() *modManager {
	once.Do(func() {
		singleton = &modManager{}
	})
	return singleton
}

func (sm *modManager) GetServer() *HttpServer {
	return sm.server
}

func (sm *modManager) SetServer(s *HttpServer) {
	sm.server = s
}

func (sm *modManager) GetRouter() *Router {
	return sm.router
}

func (sm *modManager) SetRouter(r *Router) {
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

func (sm *modManager) SortRoutes() {
	sort.SliceStable(sm.router.Routes, func(i, j int) bool {
		return len(sm.router.Routes[i].Pattern.String()) > len(sm.router.Routes[j].Pattern.String())
	})
}

func (s *HttpServer) WaitShutdown() {
	irqSig := make(chan os.Signal, 1)
	signal.Notify(irqSig, syscall.SIGINT, syscall.SIGTERM)

	//Wait interrupt or shutdown request through /shutdown
	select {
	case sig := <-irqSig:
		log.Printf("Shutdown request (signal: %v)", sig)
	case sig := <-s.shutdownReq:
		log.Printf("Shutdown request (/shutdown %v)", sig)
	}

	log.Printf("Stoping http server ...")

	//Create shutdown context with 10 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	//shutdown the server
	err := s.Shutdown(ctx)
	if err != nil {
		log.Printf("Shutdown request error: %v", err)
	}
}

func (s *HttpServer) ShutdownHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Shutdown server"))

	//Do nothing if shutdown request already issued
	//if s.reqCount == 0 then set to 1, return true otherwise false
	if !atomic.CompareAndSwapUint32(&s.reqCount, 0, 1) {
		log.Printf("Shutdown through API call in progress...")
		return
	}

	go func() {
		s.shutdownReq <- true
	}()
}

func (sm *modManager) Shutdown(w http.ResponseWriter) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := sm.server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}
	log.Println("Server exiting")
}
