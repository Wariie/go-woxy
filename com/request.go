package com

import (
	"bytes"
	"encoding/json"
	"log"

	rand "github.com/Wariie/go-woxy/tools"
)

var defaultPath = "/connect"

//IP Address
type IP string

//Port Server port
type Port string

//Path Server path
type Path string

//Protocol Server protocol
type Protocol string

/*Server - Struct */
type Server struct {
	IP       IP
	Port     Port
	Path     Path
	Protocol Protocol
}

/*Request - server request*/
type Request interface {
	Decode(b []byte)
	Encode() []byte
	Generate(list ...interface{})
	GetPath() string
	GetType() string
	GetSecret() string
}

/*ConnexionRequest - server connexion request */
type ConnexionRequest struct {
	CustomCommands []string
	ModHash        string
	Name           string
	Pid            string
	Port           string
	ResourcePath   string
	Secret         string
	Type           string
	State          string
}

//Decode - Decode JSON to ConnexionRequest
func (cr *ConnexionRequest) Decode(b []byte) {
	json.NewDecoder(bytes.NewBuffer(b)).Decode(cr)
}

//Encode - Encode ConnexionRequest to JSON
func (cr *ConnexionRequest) Encode() []byte {
	b, err := json.Marshal(cr)
	if err != nil {
		log.Println("error:", err)
	}
	return b
}

//Generate - Generate ConnexionRequest with params
func (cr *ConnexionRequest) Generate(list ...interface{}) {
	cr.CustomCommands = list[0].([]string)
	cr.ModHash = rand.String(15)
	cr.Name = list[1].(string)
	cr.Port = list[2].(string)
	cr.Pid = list[3].(string)
	cr.Secret = list[4].(string)
	cr.Type = "Connexion"

}

/*GetPath - ConnexionRequest path string*/
func (cr *ConnexionRequest) GetPath() string {
	return defaultPath
}

/*GetType - ConnexionRequest request type*/
func (cr *ConnexionRequest) GetType() string {
	return cr.Type
}

/*GetSecret - ConnexionRequest request secret*/
func (cr *ConnexionRequest) GetSecret() string {
	return cr.ModHash
}

/*CommandRequest - CommandRequest*/
type CommandRequest struct {
	Command      string
	Content      string
	Hash         string
	Name         string
	ResourcePath string
	Secret       string
	Type         string
}

//Decode - Decode JSON to CommandRequest
func (cr *CommandRequest) Decode(b []byte) {
	json.NewDecoder(bytes.NewBuffer(b)).Decode(cr)
}

//Encode - Encode CommandRequest to JSON
func (cr *CommandRequest) Encode() []byte {
	b, err := json.Marshal(cr)
	if err != nil {
		log.Println("error:", err)
	}
	return b
}

//Generate - Generate CommandRequest with params
//- Command   string
//- Hash 	  string
//- Name 	  string
//- Secret    string
func (cr *CommandRequest) Generate(list ...interface{}) {
	cr.Command = list[0].(string)
	cr.Hash = list[1].(string)
	cr.Name = list[2].(string)
	cr.Type = "Command"
	cr.Secret = list[3].(string)
}

/*GetPath - CommandRequest path string*/
func (cr *CommandRequest) GetPath() string {
	return "/cmd"
}

/*GetSecret - CommandRequest request secret*/
func (cr *CommandRequest) GetSecret() string {
	return cr.Secret
}

/*GetType - CommandRequest request type*/
func (cr *CommandRequest) GetType() string {
	return cr.Type
}

//TODO UPDATE REQUEST ?
