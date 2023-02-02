package core

import (
	"log"
	"sync"
	"time"

	com "github.com/Wariie/go-woxy/com"
)

// Supervisor -
type Supervisor struct {
	mux        sync.Mutex
	listModule []string
	core       *Core
}

// Remove -
func (s *Supervisor) Remove(m string) {
	s.mux.Lock()
	defer s.mux.Unlock()
	for i := range s.listModule {
		if m == s.listModule[i] {
			s.listModule[i] = s.listModule[len(s.listModule)-1] // Copy last element to index i.
			s.listModule[len(s.listModule)-1] = ""              // Erase last element (write zero value).
			s.listModule = s.listModule[:len(s.listModule)-1]   // Truncate slice.
			break
		}
	}
}

// Add -
func (s *Supervisor) Add(m string) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.listModule = append(s.listModule, m)
}

// Supervise -
func (s *Supervisor) Supervise() {
	//ENDLESS LOOP
	for {
		var mod *ModuleConfig

		s.mux.Lock()
		modulesList := s.listModule
		s.mux.Unlock()
		//FOR EACH REGISTERED MODULE
		for k := range modulesList {
			//CHECK MODULE RUNNING

			mod = s.core.GetModule(modulesList[k])

			timeBeforeLastPing := time.Until(mod.EXE.LastPing)

			var editStat bool = false

			//if Loading | Unknown | Online
			var managingState bool = mod.STATE < com.Downloaded && mod.STATE >= com.Unknown

			if managingState && timeBeforeLastPing.Minutes() < -5 {
				if mod.STATE != com.Unknown {
					mod.STATE = com.Unknown
					editStat = true
					log.Println("GO-WOXY Core - Module " + mod.NAME + " not pinging since 5 minutes")
				}
			} else if mod.STATE != com.Online && mod.STATE != com.Loading && mod.STATE != com.Downloaded {
				mod.STATE = com.Online
				editStat = true
			}

			if editStat {
				s.core.SaveModuleChanges(mod)
			}
		}
		time.Sleep(time.Millisecond * 100)
	}
}

// SetCore - Set core
func (s *Supervisor) SetCore(core *Core) {
	s.core = core
}
