# go-woxy
Golang reverso proxy / application server

## How to use it ?

##### Clone the source code

    git clone https://guilhem-mateo.fr/git/Wariie/go-woxy.git
    cd ./go-woxy
    
##### Edit **./cfg.yml** with your config *(or try with the default one)*

##### Build go-woxy

    go build

##### Launch go-woxy

    ./go-woxy ./cfg.yml

### Dockerfile

    git clone http://guilhem-mateo.fr/git/Wariie/go-woxy.git
    cd ./go-woxy
    docker build -t go-woxy .
    docker run -d go-woxy


## Configuration

### <a name="configexample"></a>Example 

        ---
    name: easy-go-test
    server:
      address: 0.0.0.0  
    modules: 
      mod.v0: 
        version: 1.0
        types: 'web'
        exe:
          src: 'https://guilhem-mateo.fr/git/Wariie/mod.v0'
          main: "testMod.go"
        binding:
          path: 
            - '/'
            - '/css/:file'
            - '/img/:file'
            - '/js:file'
          port: 2985  
      hook:
        version: 1.0
        types: 'bind'
        binding:
          path:
            - '/saucisse' 
          root: "./ressources/saucisse.html"


**\* (M) is module option only**

### <a name="baseconfig"></a>First declare general configuration

* **name** - (Required) server config name
* **server** - (Required) server config (See [Server Configuration](#srvconfig) below for details)
* **modules** - (Required) list of module config (See [Module Configuration](#modconfig) below for details)
* **version** - server config version


### <a name="srvconfig"></a>Server Configuration
* **address** - server address (example : 127.0.0.1, guilhem-mateo.fr)
* **port** - server port (example : 2000, 8080)
* **path** - paths to bind (See example before [Example](#configexample))
* **root** - (M) bind to **root** if no **exe**
* **protocol** - transfer protocol (supported : http, https)

### <a name="modconfig"></a>Module Configuration
* **name** - module name
* **version** - module version
* **types** - (Required) module types (supported : web, bind)
* **exe** - module executable informations (See [Module Executable Configuration](#modexeconfig))
* **binding** - (Required) server config (See [Server Configuration](#srvconfig) below for details)

### <a name="modexeconfig"></a>Module Executable Configuration
* **src** - git path of module repository
* **main** - module main filename
* **bin** - source module path

## What's a go-woxy module



