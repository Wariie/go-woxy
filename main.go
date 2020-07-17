package main

import (
	"os"

	"github.com/Wariie/go-woxy/core"
)

func main() {

	if len(os.Args) == 2 {
		core.LaunchCore(os.Args[1])
	} else {
		core.LaunchCore("")
	}
}
