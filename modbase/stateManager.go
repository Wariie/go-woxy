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

func (sm *modManager) SetState(s *http.Server) {
	sm.server = s
}

func (sm *modManager) GetRouter() *gin.Engine {
	return sm.router
}

func (sm *modManager) SetRouter(r *gin.Engine) {
	sm.router = r
}

func (sm *modManager) Shutdown() {
	ctx, cancel := context.WithCancel(nil)
	defer cancel()
	singleton.server.Shutdown(ctx)
}
