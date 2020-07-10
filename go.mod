require (
	github.com/gin-gonic/gin v1.6.3
	gopkg.in/yaml.v2 v2.3.0
	guilhem-mateo.fr/go-woxy/app/com v0.0.0-00010101000000-000000000000
	guilhem-mateo.fr/go-woxy/app/rand v0.0.0-00010101000000-000000000000 // indirect
)

module guilhem-mateo.fr/go-woxy

go 1.14

replace guilhem-mateo.fr/go-woxy/app/rand => ../go-woxy/app/rand

replace guilhem-mateo.fr/go-woxy/app/com => ../go-woxy/app/com
