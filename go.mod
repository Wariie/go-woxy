module github.com/Wariie/go-woxy

go 1.17

replace github.com/Wariie/go-woxy/core => ./core

replace github.com/Wariie/go-woxy/com => ./com

replace github.com/Wariie/go-woxy/tools => ./tools

require (
	github.com/Wariie/go-woxy/com v0.0.0
	github.com/Wariie/go-woxy/tools v0.0.0
	github.com/abbot/go-http-auth v0.4.0
	github.com/shirou/gopsutil v3.21.11+incompatible
	github.com/sirupsen/logrus v1.9.0
	gopkg.in/yaml.v2 v2.4.0
)

require (
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/stretchr/testify v1.8.1 // indirect
	github.com/tklauser/go-sysconf v0.3.11 // indirect
	github.com/tklauser/numcpus v0.6.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	golang.org/x/crypto v0.5.0 // indirect
	golang.org/x/net v0.5.0 // indirect
	golang.org/x/sys v0.4.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)
