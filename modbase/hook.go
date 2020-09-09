package modbase

import (
	com "github.com/Wariie/go-woxy/com"
	"github.com/gin-gonic/gin"
)

func cmd(c *gin.Context) {
	t, b := com.GetCustomRequestType(c.Request)

	mod := GetModManager().GetMod()

	var response string
	var err error

	//TODO ADD BOOLEAN IF WE TRUST ALL REQUESTS OR IF WE CHECK SERVER CREDENTIALS

	if t["Hash"] != mod.Hash {
		response = "Error reading module Hash"
	} else {
		switch t["Type"] {
		case "Command":
			var sr com.CommandRequest
			sr.Decode(b)

			var ir interface{}
			ir = &sr
			p := (ir).(com.Request)

			switch sr.Command {
			case "Shutdown":
				response, err = shutdown(&p, c, mod)
			case "Ping":
				response, err = ping(&p, c, mod)
			default:
				for k := range mod.CustomCommands {
					if k == sr.Command {

						response, err = mod.CustomCommands[k](&p, c, mod)
						break
					}
				}
			}
		}

	}
	if err != nil {
		response += err.Error()
	}
	c.String(200, response)
}

func shutdown(r *com.Request, c *gin.Context, mod *ModuleImpl) (string, error) {
	go GetModManager().Shutdown(c)
	return "SHUTTING DOWN " + mod.Name, nil
}

func ping(r *com.Request, c *gin.Context, mod *ModuleImpl) (string, error) {
	return mod.Name + " ALIVE", nil
}
