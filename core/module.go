package core

import (
	"encoding/base64"
	"log"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/Wariie/go-woxy/com"
	"github.com/Wariie/go-woxy/tools"
	"github.com/shirou/gopsutil/process"
)

// Download - Download module from repository ( git clone )
func (mc *ModuleConfig) Download(moduleDir string) {

	if mc.STATE != Online {
		pathSeparator := string(os.PathSeparator)
		log.Println("GO-WOXY Core - Downloading " + mc.NAME)
		var listArgs []string
		var action string
		path := ""
		if _, err := os.Stat(moduleDir + mc.NAME + pathSeparator); os.IsNotExist(err) {
			listArgs = []string{"clone", mc.EXE.SRC}
			action = "Downloaded"
			path = moduleDir
		} else {
			listArgs = []string{"pull"}
			action = "Update"
			path = moduleDir + mc.NAME + pathSeparator
		}

		cmd := exec.Command("git", listArgs...)
		cmd.Dir = path
		out, err := cmd.Output()
		log.Println("GO-WOXY Core -", action, "mod :", mc, "-", string(out), err)

		mc.EXE.BIN = moduleDir + mc.NAME + pathSeparator
		mc.STATE = Downloaded
	} else {
		log.Println("GO-WOXY Core - Error trying to download/update module while running. Stop it before")
	}
}

// GetLog - GetLog from Module
func (mc *ModuleConfig) GetLog() string {
	b, err := os.ReadFile(mc.EXE.BIN + "/log.log")
	if err != nil {
		log.Println("GO-WOXY Core - Error reading module log file : " + err.Error())
		return ""
	}
	return string(b)
}

// GetPerf - GetPerf from Module
func (mc *ModuleConfig) GetPerf() (float64, float32) {
	var p, err = process.NewProcess(int32(mc.pid))
	ram, err := p.MemoryPercent()
	cpu, err := p.Percent(0)
	name, err := p.Name()
	log.Println("PERF :", name, err)
	return cpu, ram
}

// GetServer - Get Module Server configuration
func (mc *ModuleConfig) GetServer(path string) com.Server {
	if path == "" {
		path = mc.BINDING.PATH[0].FROM
	}
	return com.Server{IP: com.IP(mc.BINDING.ADDRESS), Path: com.Path(path), Port: com.Port(mc.BINDING.PORT), Protocol: com.Protocol(mc.BINDING.PROTOCOL)}
}

// Start - Start module with config args and auto args
func (mc *ModuleConfig) Start() {
	log.Println("GO-WOXY Core - Starting mod : ", mc)

	var platformParam []string
	if runtime.GOOS == "windows" {
		platformParam = []string{"cmd", "/c ", "go", "run", mc.EXE.MAIN, "1>", "log.log", "2>&1"}
	} else {
		platformParam = []string{"/bin/sh", "-c", "go run " + mc.EXE.MAIN + " > log.log 2>&1"}
	}

	cmd := exec.Command(platformParam[0], platformParam[1:]...)
	cmd.Dir = mc.EXE.BIN
	cmd.Start()
	mc.EXE.LastPing = time.Now()
	mc.pid = cmd.Process.Pid
}

func (mc *ModuleConfig) copyAPIKey(api_key string) {
	destination, err := os.Create("." + string(os.PathSeparator) + mc.EXE.BIN + string(os.PathSeparator) + ".secret")
	if err != nil {
		log.Println("GO-WOXY Core - Error creating mod secret file : ", err)
	}

	defer destination.Close()

	nBytes, err := destination.Write([]byte(api_key))
	if err != nil {
		log.Println("GO-WOXY Core - Error Copying Secret : ", err, nBytes)
	}
}

func (mc *ModuleConfig) generateAPIKey() {
	mc.API_KEY = base64.URLEncoding.EncodeToString([]byte(tools.String(64)))
}

// ErrorPage - Content description for go-woxy error page
type ErrorPage struct {
	Title   string
	Code    int
	Message string
}

/*ModuleConfig - Module configuration */
type ModuleConfig struct {
	API_KEY      string
	AUTH         ModuleAuthConfig
	BINDING      ServerConfig
	COMMANDS     []string
	EXE          ModuleExecConfig
	NAME         string
	pid          int
	PK           string
	RESOURCEPATH string
	LOG          ModuleLogConfig
	STATE        ModuleState
	TYPES        string
	VERSION      int
}

/*ModuleLogConfig - Module Logging Configuration */
type ModuleLogConfig struct {
	Enabled *bool  `yaml:"enabled"`
	File    string `yaml:"file"`
	Path    string `yaml:"path"`
}

func (mlc *ModuleLogConfig) IsEnabled() bool {
	return *mlc.Enabled
}

/*ModuleExecConfig - Module exec file informations */
type ModuleExecConfig struct {
	BIN        string
	MAIN       string
	REMOTE     bool
	SRC        string
	SUPERVISED bool
	LastPing   time.Time
}

/*ServerConfig - Server configuration*/
type ServerConfig struct {
	ADDRESS  string
	PATH     []Route
	PORT     string
	PROTOCOL string
	ROOT     string
	CERT     string
	CERT_KEY string
}

/*ModuleAuthConfig - ModuleConfig Auth configuration*/
type ModuleAuthConfig struct {
	ENABLED bool
	TYPE    string
}

// Route - Route redirection
type Route struct {
	FROM string
	TO   string
}

// ModuleState - ModuleConfig State
type ModuleState int

// ModuleState list
const (
	Stopped    ModuleState = 0
	Unknown    ModuleState = 1
	Online     ModuleState = 2
	Downloaded ModuleState = 3
	Loading    ModuleState = 4

	Error  ModuleState = 999
	Failed ModuleState = 998
)
