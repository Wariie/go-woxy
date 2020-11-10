package core

import (
	"sync"

	"github.com/gin-gonic/gin"
)

type manager struct {
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
	return sm.config
}

func (sm *manager) SetState(c *Config) {
	sm.config = c
}

func (sm *manager) GetRouter() *gin.Engine {
	return sm.router
}

func (sm *manager) SetRouter(r *gin.Engine) {
	sm.router = r
}

func (sm *manager) GetCommandProcessor() *CommandProcessorImpl {
	return sm.cp
}

func (sm *manager) SetCommandProcessor(cp *CommandProcessorImpl) {
	sm.cp = cp
}

func (sm *manager) SetSupervisor(s *Supervisor) {
	sm.s = s
}

func (sm *manager) GetSupervisor() *Supervisor {
	return sm.s
}

func (sm *manager) SaveModuleChanges(mc *ModuleConfig) {
	sm.config.MODULES[mc.NAME] = *mc
}
