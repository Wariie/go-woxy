package modbase

import (
	"net/http"

	"github.com/Wariie/go-woxy/com"
)

func cmd(w http.ResponseWriter, r *http.Request) {
	t, b := com.GetCustomRequestType(r)

	mod := GetModManager().GetMod()

	var response string
	var err error

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
				response, err = shutdown(&p, w, r, mod)
			case "Ping":
				response, err = ping(&p, w, r, mod)
			default:
				for k := range mod.CustomCommands {
					if k == sr.Command {

						response, err = mod.CustomCommands[k](&p, w, r, mod)
						break
					}
				}
			}
		}

	}
	if err != nil {
		response += err.Error()
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(response))
}

func shutdown(r *com.Request, w http.ResponseWriter, re *http.Request, mod *ModuleImpl) (string, error) {
	go GetModManager().Shutdown(w)
	return "SHUTTING DOWN " + mod.Name, nil
}

func ping(r *com.Request, w http.ResponseWriter, re *http.Request, mod *ModuleImpl) (string, error) {
	return mod.Name + " ALIVE", nil
}
