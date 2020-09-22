# go-woxy

[![Go Report Card](https://goreportcard.com/badge/github.com/Wariie/go-woxy)](https://goreportcard.com/report/github.com/Wariie/go-woxy)
[![Build Status](https://travis-ci.com/Wariie/go-woxy.svg?branch=master)](https://travis-ci.com/Wariie/go-woxy)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2FWariie%2Fgo-woxy.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2FWariie%2Fgo-woxy?ref=badge_shield)

Golang reverse proxy / application server

## Installation

Clone the source code

    git clone https://github.com/Wariie/go-woxy.git
    cd ./go-woxy
  
Edit **cfg.yml** with your config *(or try with the default one)*

Build go-woxy

    go build

Launch go-woxy

    go-woxy cfg.yml

Dockerfile

    git clone http://github.com/Wariie/go-woxy.git
    cd ./go-woxy
    docker build -t go-woxy .
    docker run -d go-woxy

## Configuration

### Example

    ---
    name: easy-go-test
    server:
      #cert: 'ca-cert.pem'
      #cert_key: 'ca-key.pem'
    modules: 
      mod-manager:
        version: 1.0
        types: 'web'
        exe:
          remote: false
          src: 'https://github.com/Wariie/mod-manager.git'
          main: 'main.go'
          supervised: true
        binding:
          path:
            - from: '/mod-manager'
              to: '/'
          port: 2001
        auth:
          enabled: true
          type: 'http'
      mod.v0: 
        version: 1.0
        types: 'web'
        exe:
          remote: false
          src: 'https://github.com/Wariie/mod.v0.git'
          main: "testMod.go"
          supervised: true
        binding:
          path: 
            - from: '/'
          port: 2985  
      hook:
        types: 'bind'
        binding:
          path:
            - from: '/saucisse' 
          root: "./ressources/saucisse.html"
      favicon:
        types: 'bind'
        binding:
          path:
            - from: '/favicon.ico'
          root: "./ressources/favicon.ico"
  
### General configuration

* **modules** - (Required) list of module config (See [Module Configuration](#module-configuration) below for details)
* **motd** - motd filepath (default : "motd.txt")
* **name** - (Required) server config name
* **server** - (Required) server config (See [Server Configuration](#server-configuration) below for details)
* **version** - server config version

### Server Configuration

* **address** - server address (example : 127.0.0.1, guilhem-mateo.fr)
* **path** - paths to bind (from: 'path', to: 'customPath') (See example before [Example](#example))
* **port** - server port (example : 2000, 8080)
* **protocol** - transfer protocol (supported : http, https)
* **root** - (M) bind to **root** if no **exe**
* **cert** - SSL certificate path
* **cert_key** - SSL key certificate path

### Module Configuration

* **auth** - auth config (See [Module Authentication Configuration](#module-authentication-configuration) below for details)
* **binding** - (Required) server config (See [Server Configuration](#server-configuration) below for details)
* **exe** - module executable informations (See [Module Executable Configuration](#module-executable-configuration))
* **name** - (Required) module name
* **types** - (Required) module types (supported : web, bind)
* **version** - module version

### Module Executable Configuration

* **bin** - source module path
* **main** - module main filename
* **remote** - boolean if it's executed on remote server (default : false)
* **src** - git path of module repository
* **supervised** - boolean if module need to be supervised

### Module Authentication Configuration

* **enabled** - boolean for authentication activation
* **type** - authentication type

# go-woxy Module

Deploy a web-app easily and deploy it through go-woxy

## Simple example

    package main

    import (
        "log"
        "net/http"

        "github.com/gin-gonic/gin"

        modbase "github.com/Wariie/go-woxy/modbase"
    )

    func main() {
        var m modbase.ModuleImpl

        m.Name = "mod.v0"
        m.InstanceName = "mod test v0"
        m.SetServer("", "", "2985", "")
        m.Init()
        m.Register("GET", "/", index, "WEB")
        m.Run()
    }

    func index(ctx *gin.Context) {
        ctx.HTML(http.StatusAccepted, "index.html", gin.H{
            "title": "Guilhem MATEO",
        })
        log.Println("GET / mod.v0", ctx.Request.RemoteAddr)
    }

Much more **(mod-manager) [here](https://github.com/Wariie/mod-manager)**

Want to build your own ?

Check **[here](https://github.com/Wariie/go-woxy/tree/master/modbase)** for the module base code

# go-woxy API

//TODO


## License
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2FWariie%2Fgo-woxy.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2FWariie%2Fgo-woxy?ref=badge_large)