package core

import (
	"log"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	com "github.com/Wariie/go-woxy/com"
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

	for {
		mods := GetManager().GetConfig().MODULES

		for k := range s.listModule {
			m := mods[s.listModule[k]]
			if checkModuleRunning(m) {
				if m.STATE != Online && m.STATE != Loading && m.STATE != Downloaded {
					m.STATE = Online
				}
			} else {
				m.STATE = Unknown
				s.Remove(m.NAME)
			}
			GetManager().SaveModuleChanges(&m)
		}
		time.Sleep(time.Millisecond * 200)
	}
}

func checkModuleRunning(mc ModuleConfig) bool {
	try := 0
	b := false

	for b == false || try < 5 {
		if mc.pid != 0 && (mc.EXE != ModuleExecConfig{}) {
			b = checkPidRunning(&mc)
		}

		if !b {
			b = checkModulePing(&mc)
		}
		try++
	}
	return b
}

func checkModulePing(mc *ModuleConfig) bool {
	var cr com.CommandRequest
	cr.Generate("Ping", mc.PK, mc.NAME, secretHash)
	resp, err := com.SendRequest(mc.GetServer("/cmd"), &cr, false)
	if err != nil {
		return false
	} else if strings.Contains(resp, mc.NAME+" ALIVE") {
		return true
	}
	return false
}

func checkPidRunning(mc *ModuleConfig) bool {
	var platformParam []string
	var c string
	if runtime.GOOS == "windows" {
		c = "tasklist"
		platformParam = []string{"/fi", "pid eq " + strconv.Itoa(mc.pid)}
	} else {
		c = "ps -p " + strconv.Itoa(mc.pid)
		//platformParam = []string{"-p", }
	}

	cmd := exec.Command(c, platformParam...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Print("Error", err.Error())
	}

	b := false

	r := strings.Split(strings.TrimSpace(string(output)), "\n")
	if runtime.GOOS == "windows" {
		if len(r) == 3 && strings.Contains(r[2], strconv.Itoa(mc.pid)) {
			b = true
		}
	} else {
		if len(r) == 2 && strings.Contains(r[1], strconv.Itoa(mc.pid)) {
			b = true
		}
	}
	return b
}
