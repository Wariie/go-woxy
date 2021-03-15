package core

import (
	"log"
	"time"
)

//Supervisor -
type Supervisor struct {
	listModule []string
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
		//FOR EACH REGISTERED MODULE
		for k := range s.listModule {
			if k >= len(s.listModule) {
				defer s.Reload()
				return
			}
			//CHECK MODULE RUNNING
			m := GetManager().GetConfig().MODULES[s.listModule[k]]
			timeBeforeLastPing := time.Until(m.EXE.LastPing)

			if m.STATE != Loading && m.STATE != Downloaded && timeBeforeLastPing.Minutes() > 5 {
				m.STATE = Unknown
				//TODO BEST LOGGING
				log.Println("GO-WOXY Core - Module " + m.NAME + " not pinging since 5 minutes")
				s.Remove(m.NAME)
			} else if m.STATE != Online && m.STATE != Loading && m.STATE != Downloaded {
				m.STATE = Online
			}
			GetManager().SaveModuleChanges(&m)
		}
		time.Sleep(time.Millisecond * 10)
	}
}

//Reload - Reload supervisor
func (s *Supervisor) Reload() {
	defer s.Supervise()
}
