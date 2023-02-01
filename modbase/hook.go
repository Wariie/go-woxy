package modbase

import (
	"net/http"

	"github.com/Wariie/go-woxy/com"
)

func resources(path string, modResourcePath string) HandlerFunc {
	return HandlerFunc(func(ctx *Context) {
		http.StripPrefix(path+modResourcePath, http.FileServer(http.Dir("."+modResourcePath)))

	})
}

func notFound() HandlerFunc {
	return HandlerFunc(func(ctx *Context) {
		http.NotFound(ctx.ResponseWriter, ctx.Request)
	})
}

func cmd() HandlerFunc {
	return HandlerFunc(func(ctx *Context) {
		t, b := com.GetCustomRequestType(ctx.Request)

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
					response, err = shutdown(&p, ctx.ResponseWriter, ctx.Request, mod)
				case "Ping":
					response, err = ping(&p, ctx.ResponseWriter, ctx.Request, mod)
				default:
					for k := range mod.CustomCommands {
						if k == sr.Command {

							response, err = mod.CustomCommands[k](&p, ctx.ResponseWriter, ctx.Request, mod)
							break
						}
					}
				}
			}
		}
		if err != nil {
			response += err.Error()
		}
		ctx.ResponseWriter.WriteHeader(http.StatusOK)
		ctx.ResponseWriter.Write([]byte(response))
	})
}

func shutdown(r *com.Request, w http.ResponseWriter, re *http.Request, mod *ModuleImpl) (string, error) {
	go GetModManager().Shutdown(w)
	return "SHUTTING DOWN " + mod.Name, nil
}

func ping(r *com.Request, w http.ResponseWriter, re *http.Request, mod *ModuleImpl) (string, error) {
	return mod.Name + " ALIVE", nil
}
