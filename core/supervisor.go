package core

import (
	"log"
	"time"
)

//Supervisor -
type Supervisor struct {
	listModule []string
	core       *Core
}

//Remove -
func (s *Supervisor) Remove(m string) {
	for i := range s.listModule {
		if m == s.listModule[i] {
			s.listModule[i] = s.listModule[len(s.listModule)-1] // Copy last element to index i.
			s.listModule[len(s.listModule)-1] = ""              // Erase last element (write zero value).
			s.listModule = s.listModule[:len(s.listModule)-1]   // Truncate slice.
			break
		}
	}
}

//Add -
func (s *Supervisor) Add(m string) {
	s.listModule = append(s.listModule, m)
}

//Supervise -
func (s *Supervisor) Supervise() {
	//ENDLESS LOOP
	for {
		var mod *ModuleConfig
		//FOR EACH REGISTERED MODULE
		for k := range s.listModule {
			if k >= len(s.listModule) {
				defer s.Reload()
				return
			}
			//CHECK MODULE RUNNING

			for _, m := range s.core.GetConfig().modulesList {
				if m.NAME == s.listModule[k] {
					mod = m
					break
				}
			}

			timeBeforeLastPing := time.Until(mod.EXE.LastPing)

			if mod.STATE != Loading && mod.STATE != Downloaded && timeBeforeLastPing.Minutes() > 5 {
				mod.STATE = Unknown
				//TODO BEST LOGGING
				log.Println("GO-WOXY Core - Module " + mod.NAME + " not pinging since 5 minutes")
				s.Remove(mod.NAME)
			} else if mod.STATE != Online && mod.STATE != Loading && mod.STATE != Downloaded {
				mod.STATE = Online
			}

			//s.core.SaveModuleChanges(&m)
		}
		time.Sleep(time.Millisecond * 10)
	}
}

//Reload - Reload supervisor
func (s *Supervisor) Reload() {
	defer s.Supervise()
}

func (s *Supervisor) SetCore(core *Core) {
	s.core = core
}
