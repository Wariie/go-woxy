package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	com "github.com/Wariie/go-woxy/com"
)

//Command -
type Command interface {
	Run(r com.Request, m *ModuleConfig, args ...string) (string, error)
	Error() error
	GetResult() string
	registerExecutor(run func(r com.Request, m *ModuleConfig, args ...string) (string, error))
	GetName() string
}

//ModuleCommand -
type ModuleCommand struct {
	name     string
	result   string
	executor func(com.Request, *ModuleConfig, ...string) (string, error)
	err      error
}

//Run -
func (mc *ModuleCommand) Run(r com.Request, m *ModuleConfig, args ...string) (string, error) {
	return mc.executor(r, m, args...)
}

//Error -
func (mc *ModuleCommand) Error() error {
	return mc.err
}

//GetResult -
func (mc *ModuleCommand) GetResult() string {
	return mc.result
}

//GetName -
func (mc *ModuleCommand) GetName() string {
	return mc.name
}

func (mc *ModuleCommand) registerExecutor(fn func(com.Request, *ModuleConfig, ...string) (string, error)) {
	mc.executor = fn
}

//CommandProcessor -
type CommandProcessor interface {
	Register(name string, run func(com.Request, *ModuleConfig, ...string) (string, error)) bool
	Run(name string, r com.Request, m ModuleConfig, args ...string) Command
}

//CommandProcessorImpl -
type CommandProcessorImpl struct {
	commands []Command
}

//Register - Register new ModuleCommand in CommandProcessorImpl
func (cp *CommandProcessorImpl) Register(name string, run func(com.Request, *ModuleConfig, ...string) (string, error)) {
	c := ModuleCommand{name: name}
	c.registerExecutor(run)
	cp.commands = append(cp.commands, &c)
}

//Run - Run Command in CommandProcessorImpl
func (cp *CommandProcessorImpl) Run(name string, r com.Request, m *ModuleConfig, args ...string) (string, error) {
	for k := range cp.commands {
		if cp.commands[k].GetName() == name {
			c := cp.commands[k]
			return c.Run(r, m, args...)
		}
	}
	return "Error : Command not found", nil
}

//Init - CommandProcessorImpl
func (cp *CommandProcessorImpl) Init() {
	cp.Register("List", listModuleCommand)
	cp.Register("Log", logModuleCommand)
	cp.Register("Performance", performanceModuleCommand)
	cp.Register("Ping", defaultForwardCommand)
	cp.Register("Restart", restartModuleCommand)
	cp.Register("Shutdown", defaultForwardCommand)
	cp.Register("Start", startModuleCommand)
}

func defaultForwardCommand(r com.Request, mc *ModuleConfig, args ...string) (string, error) {
	return com.SendRequest(mc.GetServer("/cmd"), r, false)
}

func listModuleCommand(r com.Request, mc *ModuleConfig, args ...string) (string, error) {
	rb, err := json.Marshal(GetManager().GetConfig().MODULES)
	if err != nil {
		return "Error :", err
	}
	return string(rb), nil
}

func logModuleCommand(r com.Request, mc *ModuleConfig, args ...string) (string, error) {
	return mc.GetLog(), nil
}

func performanceModuleCommand(r com.Request, mc *ModuleConfig, args ...string) (string, error) {
	c, ra := mc.GetPerf()
	return "CPU/RAM : " + fmt.Sprintf("%f", c) + "/" + fmt.Sprintf("%f", ra), nil
}

func restartModuleCommand(r com.Request, mc *ModuleConfig, args ...string) (string, error) {
	response := ""
	cr := (r).(*com.CommandRequest)
	cr.Command = "Shutdown"
	rqtS, err := com.SendRequest(mc.GetServer("/cmd"), cr, false)
	if strings.Contains(rqtS, "SHUTTING DOWN "+mc.NAME) || (err != nil && strings.Contains(err.Error(), "An existing connection was forcibly closed by the remote host")) {
		time.Sleep(10 * time.Second)
		if err := mc.Setup(GetManager().GetRouter(), false); err != nil {
			response += "Error :" + err.Error()
			log.Println(err)
		} else {
			response += "Success"
			mc.STATE = Stopped
			GetManager().SaveModuleChanges(mc)
		}
	} else {
		response += "Error :" + rqtS
		if err != nil {
			response += " - " + err.Error()
		}
	}
	return response, err
}

func startModuleCommand(r com.Request, mc *ModuleConfig, args ...string) (string, error) {
	response := ""
	mods := GetManager().GetConfig().MODULES
	var mo *ModuleConfig
	for m := range mods {
		if mods[m].NAME == (r).(*com.CommandRequest).Content {
			*mo = mods[m]
		}
	}

	var err error
	if mo.STATE != Online {
		err = mo.Setup(GetManager().GetRouter(), false)
		if err == nil {
			response += "Success"
			mc.STATE = Stopped
			GetManager().SaveModuleChanges(mo)
		}
	} else {
		err = errors.New("Module already Online")
	}
	return response, err
}