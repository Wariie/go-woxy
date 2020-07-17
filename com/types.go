package com

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

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
}

/*ConnexionRequest - server connexion request */
type ConnexionRequest struct {
	Name    string
	Secret  string
	Port    string
	ModHash string
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
	cr.Secret = list[1]
	cr.Port = list[2]
	cr.ModHash = rand.String(15)
}

/*GetPath - ConnexionRequest path string*/
func (cr *ConnexionRequest) GetPath() string {
	return defaultPath
}

//ConnexionReponseRequest - ConnexionReponseRequest
type ConnexionReponseRequest struct {
	Name  string
	State string
	Hash  string
	Port  string
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
	cr.Name = list[0]
	cr.State = list[1]
	cr.Hash = list[2]
	cr.Port = list[3]
}

/*GetPath - ConnexionReponseRequest path string*/
func (cr *ConnexionReponseRequest) GetPath() string {
	return defaultPath
}

/*ShutdownRequest - server connexion request */
type ShutdownRequest struct {
	Name string
	Hash string
}

//Decode - Decode JSON to ShutdownRequest
func (cr *ShutdownRequest) Decode(b []byte) {
	json.NewDecoder(bytes.NewBuffer(b)).Decode(cr)
}

//Encode - Encode ShutdownRequest to JSON
func (cr *ShutdownRequest) Encode() []byte {
	b, err := json.Marshal(cr)
	if err != nil {
		log.Println("error:", err)
	}
	return b
}

//Generate - Generate ConnexionRequest with params
func (cr *ShutdownRequest) Generate(list ...string) {
	cr.Name = list[0]
	cr.Hash = list[1]
}

/*GetPath - ShutdownRequest path string*/
func (cr *ShutdownRequest) GetPath() string {
	return "/shutdown"
}

//SendRequest - sens request to server
func SendRequest(s Server, r Request, loging bool) string {

	if loging {
		fmt.Println("LAUNCH REQUEST - ", r, " TO ", s)
	}

	var customPath string = defaultPath
	if r.GetPath() != "" {
		customPath = r.GetPath()
	}

	var url string = s.Protocol + "://" + s.IP + ":" + s.Port + s.Path + customPath

	//SEND REQUEST
	resp, err := http.Post(url, "text/json", bytes.NewBuffer(r.Encode()))
	if err != nil {
		log.Println(err)
	}
	//defer resp.Body.Close()
	if resp != nil && resp.Body != nil {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		return buf.String()
	}
	return ""
}
