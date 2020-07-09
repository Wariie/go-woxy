require (
	github.com/gin-gonic/gin v1.6.3
	gopkg.in/yaml.v2 v2.3.0
	guilhem-mateo.fr/git/Wariie/website.git v0.0.0-20200625000136-11cc54a6c423 // indirect
	guilhem-mateo.fr/testgo/app/com v0.0.0-00010101000000-000000000000
	guilhem-mateo.fr/testgo/app/rand v0.0.0-00010101000000-000000000000 // indirect
)

module guilhem-mateo.fr/testgo

go 1.14

replace guilhem-mateo.fr/testgo/app/rand => ../testgo/app/rand

replace guilhem-mateo.fr/testgo/app/com => ../testgo/app/com
