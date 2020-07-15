package main

import (
	"os"

	"guilhem-mateo.fr/git/Wariie/go-woxy.git/app"
)

func main() {

	if len(os.Args) == 2 {
		app.LaunchCore(os.Args[1])
	} else {
		app.LaunchCore("")
	}
}
