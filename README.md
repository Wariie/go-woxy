
# go-woxy

Golang reverso proxy / application server

## How to use it

### Installation

#### Clone the source code

    git clone https://github.com/Wariie/go-woxy.git
    cd ./go-woxy
  
#### Edit **./cfg.yml** with your config *(or try with the default one)*

#### Build go-woxy

    go build

#### Launch go-woxy

    ./go-woxy ./cfg.yml

### Dockerfile

    git clone http://github.com/Wariie/go-woxy.git
    cd ./go-woxy
    docker build -t go-woxy .
    docker run -d go-woxy

### Configuration

#### Example  

        ---
    name: easy-go-test
    server:
      address: 0.0.0.0  
    modules: 
      mod.v0: 
        version: 1.0
        types: 'web'
        exe:
          src: 'https://github.com/Wariie/mod.v0'
          main: "testMod.go"
        binding:
          path: 
            - from: '/'
          port: 2985  
      hook:
        version: 1.0
        types: 'bind'
        binding:
          path:
            - from: '/saucisse' 
          root: "./ressources/saucisse.html"
  
##### **(M) is module option only**
  
#### General configuration

* **name** - (Required) server config name
* **server** - (Required) server config (See [Server Configuration](#server-configuration) below for details)
* **modules** - (Required) list of module config (See [Module Configuration](#module-configuration) below for details)
* **version** - server config version

#### Server Configuration

* **address** - server address (example : 127.0.0.1, guilhem-mateo.fr)
* **port** - server port (example : 2000, 8080)
* **path** - paths to bind (from: 'path', to: 'customPath') (See example before [Example](#example))
* **root** - (M) bind to **root** if no **exe**
* **protocol** - transfer protocol (supported : http, https)

#### Module Configuration

* **name** - (Required) module name
* **version** - module version
* **types** - (Required) module types (supported : web, bind)
* **exe** - module executable informations (See [Module Executable Configuration](#module-executable-configuration))
* **binding** - (Required) server config (See [Server Configuration](#server-configuration) below for details)

#### Module Executable Configuration

* **src** - git path of module repository
* **main** - module main filename
* **bin** - source module path

### What's a go-woxy module

Want to build your own ?
See an example right **[there](https://github.com/Wariie/mod.v0)**
  
Check **[here](https://github.com/Wariie/go-woxy/tree/master/modbase)** for the module base code
