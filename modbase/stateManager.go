package modbase

import (
	"context"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

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

func (sm *modManager) Shutdown(c *gin.Context) {
	ctx, cancel := context.WithCancel(c)
	defer cancel()
	singleton.server.Shutdown(ctx)
}
