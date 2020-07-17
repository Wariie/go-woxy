package main

import (
	"os"

	"github.com/Wariie/go-woxy/app"
)

func main() {

	if len(os.Args) == 2 {
		app.LaunchCore(os.Args[1])
	} else {
		app.LaunchCore("")
	}
}
