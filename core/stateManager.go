package core

import (
	"bufio"
	"io"
	"os"
	"sync"

	"github.com/gorilla/mux"
)

type manager struct {
	mux           sync.Mutex
	config        *Config
	router        *mux.Router
	cp            *CommandProcessorImpl
	s             *Supervisor
	roles         []Role
	accessLogFile *os.File
}

var singleton *manager
var once sync.Once

//GetManager -
func GetManager() *manager {
	once.Do(func() {
		singleton = &manager{}
	})
	return singleton
}

func (sm *manager) GetConfig() *Config {
	sm.mux.Lock()
	defer sm.mux.Unlock()
	return sm.config
}

func (sm *manager) SetConfig(c *Config) {
	sm.mux.Lock()
	defer sm.mux.Unlock()
	sm.config = c
}

func (sm *manager) GetRouter() *mux.Router {
	sm.mux.Lock()
	defer sm.mux.Unlock()
	return sm.router
}

func (sm *manager) SetRouter(r *mux.Router) {
	sm.mux.Lock()
	defer sm.mux.Unlock()
	sm.router = r
}

func (sm *manager) GetCommandProcessor() *CommandProcessorImpl {
	sm.mux.Lock()
	defer sm.mux.Unlock()
	return sm.cp
}

func (sm *manager) SetCommandProcessor(cp *CommandProcessorImpl) {
	sm.mux.Lock()
	defer sm.mux.Unlock()
	sm.cp = cp
}

func (sm *manager) SetSupervisor(s *Supervisor) {
	sm.mux.Lock()
	defer sm.mux.Unlock()
	sm.s = s
}

func (sm *manager) GetSupervisor() *Supervisor {
	sm.mux.Lock()
	defer sm.mux.Unlock()
	return sm.s
}

func (sm *manager) AddModuleToSupervisor(mc *ModuleConfig) {
	sm.mux.Lock()
	defer sm.mux.Unlock()
	sm.s.Add(mc.NAME)
}

func (sm *manager) GetModule(name string) ModuleConfig {
	sm.mux.Lock()
	defer sm.mux.Unlock()
	return sm.config.MODULES[name]
}

func (sm *manager) SaveModuleChanges(mc *ModuleConfig) {
	sm.mux.Lock()
	defer sm.mux.Unlock()
	sm.config.MODULES[mc.NAME] = *mc
}

func (sm *manager) SearchModWithHash(hash string) ModuleConfig {
	sm.mux.Lock()
	defer sm.mux.Unlock()
	mods := sm.config.MODULES
	for i := range mods {
		if mods[i].PK == hash {
			return mods[i]
		}
	}
	return ModuleConfig{NAME: "error"}
}

func (sm *manager) GetAccessLogFileWriter() io.Writer {
	return bufio.NewWriter(sm.accessLogFile)
}

func (sm *manager) SetAccessLogFile(accesslogfile *os.File) {
	sm.accessLogFile = accesslogfile
}
