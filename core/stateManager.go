package core

import (
	"sync"

	"github.com/gin-gonic/gin"
)

type manager struct {
	mux    sync.Mutex
	config *Config
	router *gin.Engine
	cp     *CommandProcessorImpl
	s      *Supervisor
	roles  []Role
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

func (sm *manager) GetRouter() *gin.Engine {
	sm.mux.Lock()
	defer sm.mux.Unlock()
	return sm.router
}

func (sm *manager) SetRouter(r *gin.Engine) {
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
