package main

import (
	"os"

	"guilhem-mateo.fr/testgo/app"
)

func main() {

	if len(os.Args) == 2 {
		app.LaunchCore(os.Args[1])
	} else {
		app.LaunchCore("")
	}
}
