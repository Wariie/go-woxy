package com

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// SendRequest - send request to server
func SendRequest(s Server, r Request, loging bool) (string, error) {

	if loging {
		fmt.Println("LAUNCH REQUEST - ", r, " TO ", s)
	}

	var rPath = r.GetPath()
	var customPath = defaultPath
	if rPath != "" {
		if s.Path == "/" || (string(s.Path) == rPath) {
			customPath = rPath
		} else {
			customPath = string(s.Path) + rPath
		}
	}

	var url = string(s.Protocol) + "://" + string(s.IP) + ":" + string(s.Port) + customPath

	//SEND REQUEST
	resp, err := http.Post(url, "text/json", bytes.NewBuffer(r.Encode()))
	if err != nil {
		log.Println(err)
	}

	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		return buf.String(), err
	}
	return "", err
}

// GetCustomRequestType - get custom request from HTTP Request Body
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
