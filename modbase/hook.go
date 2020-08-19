package modbase

import (
	"log"

	com "github.com/Wariie/go-woxy/com"
	"github.com/gin-gonic/gin"
)

func cmd(c *gin.Context) {
	log.Println("Command request")
	t, b := com.GetCustomRequestType(c.Request)

	var response string

	if t["Hash"] != GetModManager().GetMod().Hash {
		response = "Error reading module Hash"
	} else {
		switch t["Type"] {
		case "Command":
			var sr com.CommandRequest
			sr.Decode(b)
			log.Println("Request Content - ", sr)
			switch sr.Command {
			case "Shutdown":
				response = "SHUTTING DOWN " + GetModManager().GetMod().Name
				go GetModManager().Shutdown(c)
			}
		}

	}
	c.String(200, response)
}
