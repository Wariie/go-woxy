module github.com/Wariie/go-woxy

go 1.14

replace github.com/Wariie/go-woxy/core => ./core

replace github.com/Wariie/go-woxy/com => ./com

replace github.com/Wariie/go-woxy/tools => ./tools

require (
	github.com/Wariie/go-woxy/com v0.0.0-20200721103644-c8e09825c6ea
	github.com/Wariie/go-woxy/modbase v0.0.0-20200721133627-33f0cea0d168 // indirect
	github.com/Wariie/go-woxy/tools v0.0.0 // indirect
	github.com/gin-gonic/gin v1.6.3
	gopkg.in/yaml.v2 v2.3.0
)
