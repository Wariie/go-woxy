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
	GetType() string
}

/*ConnexionRequest - server connexion request */
type ConnexionRequest struct {
	Name    string
	Secret  string
	Port    string
	ModHash string
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
	cr.Secret = list[1]
	cr.Port = list[2]
	cr.ModHash = rand.String(15)
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

//ConnexionReponseRequest - ConnexionReponseRequest
type ConnexionReponseRequest struct {
	Name  string
	State string
	Hash  string
	Port  string
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
	cr.Name = list[0]
	cr.State = list[1]
	cr.Hash = list[2]
	cr.Port = list[3]
	cr.Type = "ConnexionResponse"
}

/*GetPath - ConnexionReponseRequest path string*/
func (cr *ConnexionReponseRequest) GetPath() string {
	return defaultPath
}

/*GetType - ConnexionResponseRequest request type*/
func (cr *ConnexionReponseRequest) GetType() string {
	return cr.Type
}

/*CommandRequest - CommandRequest*/
type CommandRequest struct {
	Name    string
	Hash    string
	Type    string
	Content string
	Command string
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
	cr.Name = list[0]
	cr.Hash = list[1]
	cr.Type = "Command"
	cr.Command = list[2]
}

/*GetPath - CommandRequest path string*/
func (cr *CommandRequest) GetPath() string {
	return "cmd"
}

/*GetType - CommandRequest request type*/
func (cr *CommandRequest) GetType() string {
	return cr.Type
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

// GetCustomRequestType - get custom request from gin Request Body
func GetCustomRequestType(gRqt *http.Request) (map[string]string, []byte) {

	buf := new(bytes.Buffer)
	buf.ReadFrom(gRqt.Body)

	c := make(map[string]string)

	// unmarschal JSON
	e := json.Unmarshal(buf.Bytes(), &c)

	if e != nil {
		return map[string]string{"error": "error"}, nil
	}

	return c, buf.Bytes()
}
