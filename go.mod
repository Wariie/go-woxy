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
	github.com/go-ole/go-ole v1.2.5 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/shirou/gopsutil v3.21.3+incompatible
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.7.0 // indirect
	github.com/tklauser/go-sysconf v0.3.5 // indirect
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2 // indirect
	golang.org/x/net v0.0.0-20210331212208-0fccb6fa2b5c // indirect
	golang.org/x/sys v0.0.0-20210403161142-5e06dd20ab57 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)
