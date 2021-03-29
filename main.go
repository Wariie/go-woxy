package main

import (
	"os"

	"github.com/Wariie/go-woxy/core"
)

func main() {
	core := core.Core{}
	if len(os.Args) == 2 {
		core.GoWoxy(os.Args[1])
	} else {
		core.GoWoxy("")
	}
}
