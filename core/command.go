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
	Run(r *com.Request, m *ModuleConfig, args ...string) (string, error)
	Error() error
	GetResult() string
	registerExecutor(run func(r *com.Request, m *ModuleConfig, args ...string) (string, error))
	GetName() string
}

//ModuleCommand - Command implementation
type ModuleCommand struct {
	name     string
	result   string
	executor func(*com.Request, *ModuleConfig, ...string) (string, error)
	err      error
}

//Run - Command
func (mc *ModuleCommand) Run(r *com.Request, m *ModuleConfig, args ...string) (string, error) {
	return mc.executor(r, m, args...)
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

func (mc *ModuleCommand) registerExecutor(fn func(*com.Request, *ModuleConfig, ...string) (string, error)) {
	mc.executor = fn
}

//CommandProcessor - CommandProcessor
type CommandProcessor interface {
	Register(name string, run func(*com.Request, *ModuleConfig, ...string) (string, error)) bool
	Run(name string, r com.Request, m ModuleConfig, args ...string) Command
}

//CommandProcessorImpl -
type CommandProcessorImpl struct {
	commands []Command
}

//Register - Register new ModuleCommand in CommandProcessorImpl
func (cp *CommandProcessorImpl) Register(name string, run func(*com.Request, *ModuleConfig, ...string) (string, error)) {
	cp.register(name, run)
}

func (cp *CommandProcessorImpl) register(name string, run func(*com.Request, *ModuleConfig, ...string) (string, error)) {
	c := ModuleCommand{name: name}
	c.registerExecutor(run)
	cp.commands = append(cp.commands, &c)
}

//Run - Run command in CommandProcessorImpl
func (cp *CommandProcessorImpl) Run(name string, r *com.Request, m *ModuleConfig, args ...string) (string, error) {
	for k := range cp.commands { //DEFAULT SERVER COMMANDS
		if cp.commands[k].GetName() == name {
			return cp.commands[k].Run(r, m, args...)
		}
	}

	//PROCESS MODULE CUSTOM COMMANDS
	if m.NAME != "hub" {
		for k := range m.COMMANDS {
			if m.COMMANDS[k] == name {
				return defaultForwardCommand(r, m, args...)
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

func defaultForwardCommand(r *com.Request, mc *ModuleConfig, args ...string) (string, error) {
	return com.SendRequest(mc.GetServer("/cmd"), *r, false)
}

func pingCommand(r *com.Request, mc *ModuleConfig, args ...string) (string, error) {
	if mc.NAME != "hub" {
		return com.SendRequest(mc.GetServer("/cmd"), *r, false)
	}
	return "Pong", nil
}

func listModuleCommand(r *com.Request, mc *ModuleConfig, args ...string) (string, error) {
	rb, err := json.Marshal(GetManager().GetConfig().MODULES)
	if err != nil {
		return "Error :", err
	}
	return string(rb), nil
}

func logModuleCommand(r *com.Request, mc *ModuleConfig, args ...string) (string, error) {
	return mc.GetLog(), nil
}

func shutdownModuleCommand(r *com.Request, mc *ModuleConfig, args ...string) (string, error) {
	var response string
	var err error
	if mc.NAME != "hub" {
		response, err := defaultForwardCommand(r, mc, args...)
		if strings.Contains(response, "SHUTTING DOWN "+mc.NAME) || (err != nil && strings.Contains(err.Error(), "An existing connection was forcibly closed by the remote host")) {
			response = "Success"
			mc.STATE = Stopped
			GetManager().GetSupervisor().Remove(mc.NAME)
		}
	} else {
		response = "GO-WOXY Core - Stopping"
		go func() {
			GetManager().GetServer().shutdownReq <- true
		}()
	}
	return response, err
}

func performanceModuleCommand(r *com.Request, mc *ModuleConfig, args ...string) (string, error) {
	c, ra := mc.GetPerf()
	return "CPU/RAM : " + fmt.Sprintf("%f", c) + "/" + fmt.Sprintf("%f", ra), nil
}

func restartModuleCommand(r *com.Request, mc *ModuleConfig, args ...string) (string, error) {
	response := ""
	cr := (*r).(*com.CommandRequest)
	cr.Command = "Shutdown"
	rqtS, err := com.SendRequest(mc.GetServer("/cmd"), cr, false)
	if strings.Contains(rqtS, "SHUTTING DOWN "+mc.NAME) || (err != nil && strings.Contains(err.Error(), "An existing connection was forcibly closed by the remote host")) {
		if mc.pid != 0 {
			for mc.checkModuleRunning() {
				time.Sleep(time.Second)
			}
		}
		mc.STATE = Stopped
		if err := mc.Setup(GetManager().GetRouter(), false, GetManager().GetConfig().MODDIR); err == nil {
			response += "Success"
			mc.STATE = Stopped
		}
	} else {
		response += "Error :" + rqtS
	}
	return response, err
}

func startModuleCommand(r *com.Request, mc *ModuleConfig, args ...string) (string, error) {
	response := ""
	mods := GetManager().GetConfig().MODULES
	var mo ModuleConfig
	c := (*r).(*com.CommandRequest).Content
	for m := range mods {
		if m == c {
			mo = mods[m]
			break
		}
	}

	var err error
	if mo.STATE != Online {
		err = mo.Setup(GetManager().GetRouter(), false, GetManager().GetConfig().MODDIR)
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
