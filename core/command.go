package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	com "github.com/Wariie/go-woxy/com"
)

//Command - Command interface
type Command interface {
	Run(core *Core, r *com.Request, m *ModuleConfig, args ...string) (string, error)
	Error() error
	GetResult() string
	registerExecutor(run func(core *Core, r *com.Request, m *ModuleConfig, args ...string) (string, error))
	GetName() string
}

//ModuleCommand - Command implementation
type ModuleCommand struct {
	name     string
	result   string
	executor func(*Core, *com.Request, *ModuleConfig, ...string) (string, error)
	err      error
}

//Run - Command
func (mc *ModuleCommand) Run(core *Core, r *com.Request, m *ModuleConfig, args ...string) (string, error) {
	return mc.executor(core, r, m, args...)
}

//Error - Get command execution error
func (mc *ModuleCommand) Error() error {
	return mc.err
}

//GetResult - Get command result
func (mc *ModuleCommand) GetResult() string {
	return mc.result
}

//GetName - Get command name
func (mc *ModuleCommand) GetName() string {
	return mc.name
}

func (mc *ModuleCommand) registerExecutor(fn func(*Core, *com.Request, *ModuleConfig, ...string) (string, error)) {
	mc.executor = fn
}

//CommandProcessor - CommandProcessor
type CommandProcessor interface {
	Register(name string, run func(*Core, *com.Request, *ModuleConfig, ...string) (string, error)) bool
	Run(name string, r com.Request, m ModuleConfig, args ...string) Command
}

//CommandProcessorImpl -
type CommandProcessorImpl struct {
	commands []Command
}

//Register - Register new ModuleCommand in CommandProcessorImpl
func (cp *CommandProcessorImpl) Register(name string, run func(*Core, *com.Request, *ModuleConfig, ...string) (string, error)) {
	cp.register(name, run)
}

func (cp *CommandProcessorImpl) register(name string, run func(*Core, *com.Request, *ModuleConfig, ...string) (string, error)) {
	c := ModuleCommand{name: name}
	c.registerExecutor(run)
	cp.commands = append(cp.commands, &c)
}

//Run - Run command in CommandProcessorImpl
func (cp *CommandProcessorImpl) Run(name string, core *Core, r *com.Request, m *ModuleConfig, args ...string) (string, error) {
	for k := range cp.commands { //DEFAULT SERVER COMMANDS
		if cp.commands[k].GetName() == name {
			return cp.commands[k].Run(core, r, m, args...)
		}
	}

	//PROCESS MODULE CUSTOM COMMANDS
	if m.NAME != "hub" {
		for k := range m.COMMANDS {
			if m.COMMANDS[k] == name {
				return defaultForwardCommand(core, r, m, args...)
			}
		}
	}

	return "", errors.New("command not found")
}

//Init - Init CommandProcessorImpl with default commands
func (cp *CommandProcessorImpl) Init() {
	cp.Register("List", listModuleCommand)
	cp.Register("Log", logModuleCommand)
	cp.Register("Performance", performanceModuleCommand)
	cp.Register("Ping", pingCommand)
	cp.Register("Restart", restartModuleCommand)
	cp.Register("Shutdown", shutdownModuleCommand)
	cp.Register("Start", startModuleCommand)
}

/* ---------------------------DEFAULT COMMANDS----------------------------*/

func defaultForwardCommand(core *Core, r *com.Request, mc *ModuleConfig, args ...string) (string, error) {
	return com.SendRequest(mc.GetServer("/cmd"), *r, false)
}

func pingCommand(core *Core, r *com.Request, mc *ModuleConfig, args ...string) (string, error) {
	if mc.NAME != "hub" {
		return com.SendRequest(mc.GetServer("/cmd"), *r, false)
	}

	mc.EXE.LastPing = time.Now()

	return "Pong", nil
}

func listModuleCommand(core *Core, r *com.Request, mc *ModuleConfig, args ...string) (string, error) {
	rb, err := json.Marshal(core.GetConfig().MODULES)
	if err != nil {
		return "Error :", err
	}
	return string(rb), nil
}

func logModuleCommand(core *Core, r *com.Request, mc *ModuleConfig, args ...string) (string, error) {
	return mc.GetLog(), nil
}

func shutdownModuleCommand(core *Core, r *com.Request, mc *ModuleConfig, args ...string) (string, error) {
	var response string
	var err error
	if mc.NAME != "hub" {
		response, err := defaultForwardCommand(core, r, mc, args...)
		if strings.Contains(response, "SHUTTING DOWN "+mc.NAME) || (err != nil && strings.Contains(err.Error(), "An existing connection was forcibly closed by the remote host")) {
			response = "Success"
			mc.STATE = Stopped
			core.GetSupervisor().Remove(mc.NAME)
		} else {
			response += " " + err.Error()
		}
	} else {
		response = "GO-WOXY Core - Stopping"
		go func() {
			core.GetServer().shutdownReq <- true
		}()
	}
	return response, err
}

func performanceModuleCommand(core *Core, r *com.Request, mc *ModuleConfig, args ...string) (string, error) {
	c, ra := mc.GetPerf()
	return "CPU/RAM : " + fmt.Sprintf("%f", c) + "/" + fmt.Sprintf("%f", ra), nil
}

func restartModuleCommand(core *Core, r *com.Request, mc *ModuleConfig, args ...string) (string, error) {
	response := ""
	cr := (*r).(*com.CommandRequest)
	cr.Command = "Shutdown"
	rqtS, err := com.SendRequest(mc.GetServer("/cmd"), cr, false)
	if strings.Contains(rqtS, "SHUTTING DOWN "+mc.NAME) || (err != nil && strings.Contains(err.Error(), "An existing connection was forcibly closed by the remote host")) {

		cr.Command = "Ping"
		rqtS, err = com.SendRequest(mc.GetServer("/cmd"), cr, false)
		for strings.Contains(rqtS, "ALIVE"+mc.NAME) {
			time.Sleep(time.Second)
		}

		mc.STATE = Stopped
		if err := core.Setup(mc, false, core.GetConfig().MODDIR); err == nil {
			response += "Success"
			mc.STATE = Stopped
		}

	} else {
		response += "Error :" + rqtS
	}
	return response, err
}

func startModuleCommand(core *Core, r *com.Request, mc *ModuleConfig, args ...string) (string, error) {
	response := ""

	var err error
	if mc.STATE != Online {
		err = core.Setup(mc, false, core.GetConfig().MODDIR)
		if err == nil {
			response += "Success"
		} else {
			response += err.Error()
		}
	} else {
		err = errors.New("module already online")
	}
	return response, err
}
