module github.com/Wariie/go-woxy

go 1.15

replace github.com/Wariie/go-woxy/core => ./core

replace github.com/Wariie/go-woxy/com => ./com

replace github.com/Wariie/go-woxy/tools => ./tools

require (
	github.com/StackExchange/wmi v0.0.0-20210224194228-fe8f1750fd46 // indirect
	github.com/Wariie/go-woxy/com v0.0.0
	github.com/Wariie/go-woxy/tools v0.0.0
	github.com/abbot/go-http-auth v0.4.0
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-ole/go-ole v1.2.5 // indirect
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/kr/text v0.2.0 // indirect
	github.com/shirou/gopsutil v3.21.2+incompatible
	github.com/stretchr/testify v1.7.0 // indirect
	github.com/tklauser/go-sysconf v0.3.4 // indirect
	golang.org/x/crypto v0.0.0-20210220033148-5ea612d1eb83 // indirect
	golang.org/x/net v0.0.0-20210226172049-e18ecbb05110 // indirect
	golang.org/x/sys v0.0.0-20210313110737-8e9fff1a3a18 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)
