package com

import (
	"bytes"
	"encoding/json"
	"log"

	rand "github.com/Wariie/go-woxy/tools"
)

var defaultPath = "/connect"

/*Server - Struct */
type Server struct {
	IP       string
	Port     string
	Path     string
	Protocol string
}

/*Request - server request*/
type Request interface {
	Decode(b []byte)
	Encode() []byte
	Generate(list ...string)
	GetPath() string
	GetType() string
	GetSecret() string
}

/*ConnexionRequest - server connexion request */
type ConnexionRequest struct {
	ModHash string
	Name    string
	Pid     string
	Port    string
	Secret  string
	Type    string
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
func (cr *ConnexionRequest) Generate(list ...string) {
	cr.Name = list[0]
	cr.ModHash = rand.String(15)
	cr.Port = list[1]
	cr.Pid = list[2]
	cr.Secret = list[3]
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

//ConnexionReponseRequest - ConnexionReponseRequest
type ConnexionReponseRequest struct {
	Hash  string
	Name  string
	Port  string
	State string
	Type  string
}

//Decode - Decode JSON to ConnexionReponseRequest
func (cr *ConnexionReponseRequest) Decode(b []byte) {
	json.NewDecoder(bytes.NewBuffer(b)).Decode(cr)
}

//Encode - Encode ConnexionReponseRequest to JSON
func (cr *ConnexionReponseRequest) Encode() []byte {
	b, err := json.Marshal(cr)
	if err != nil {
		log.Println("error:", err)
	}
	return b
}

//Generate - Generate ConnexionReponseRequest with params
func (cr *ConnexionReponseRequest) Generate(list ...string) {
	cr.Hash = list[0]
	cr.Name = list[1]
	cr.Port = list[2]
	cr.State = list[3]
	cr.Type = "ConnexionResponse"
}

/*GetPath - ConnexionReponseRequest path string*/
func (cr *ConnexionReponseRequest) GetPath() string {
	return defaultPath
}

/*GetSecret - ConnexionReponseRequest request secret*/
func (cr *ConnexionReponseRequest) GetSecret() string {
	return cr.Hash
}

/*GetType - ConnexionResponseRequest request type*/
func (cr *ConnexionReponseRequest) GetType() string {
	return cr.Type
}

/*CommandRequest - CommandRequest*/
type CommandRequest struct {
	Command string
	Content string
	Hash    string
	Name    string
	Secret  string
	Type    string
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
//- Name 	  string
//- Hash 	  string
//- Command string
func (cr *CommandRequest) Generate(list ...string) {
	cr.Command = list[0]
	cr.Hash = list[1]
	cr.Name = list[2]
	cr.Type = "Command"
	cr.Secret = list[3]
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
