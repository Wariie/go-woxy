package core

import (
	"errors"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strconv"
	"strings"
	"text/template"

	auth "github.com/abbot/go-http-auth"
)

// ReverseProxyAuth - Authentication middleware
func ReverseProxyAuth(a *auth.BasicAuth, modName string, r Route) HandlerFunc {
	return HandlerFunc(func(ctx *Context) {
		user := a.CheckAuth(ctx.Request)
		if user == "" {
			ctx.ResponseWriter.Header().Add("WWW-Authenticate", "Basic realm="+strconv.Quote(a.Realm))
			ctx.ResponseWriter.WriteHeader(http.StatusUnauthorized)

		}
		ctx.ResponseWriter.Header().Set("user", user)
		ReverseProxy().Handle(ctx)
	})
}

// ReverseProxyFix - reverse proxy for mod
func ReverseProxy() HandlerFunc {
	return HandlerFunc(func(ctx *Context) {

		path := ctx.URL.Path

		mod := ctx.ModuleConfig
		route := ctx.Route

		//CHECK IF MODULE IS ONLINE
		if mod != nil && mod.STATE == Online {

			//IF ROOT IS PRESENT REDIRECT TO IT
			if strings.Contains(mod.TYPES, "bind") && mod.BINDING.ROOT != "" {
				http.ServeFile(ctx.ResponseWriter, ctx.Request, mod.BINDING.ROOT)

				//ELSE IF BINDING IS TYPE **REVERSE**
			} else if strings.Contains(mod.TYPES, "reverse") {

				if route.FROM != route.TO {
					if route.FROM != "/" {
						i := strings.Index(path, route.FROM)
						path = path[i+len(route.FROM):]
					} else {
						log.Println(path)
					}

					if route.TO != "/" && len(route.TO) > 1 && !strings.Contains(path, route.TO) {
						path = route.TO + path
					}
				}

				//BUILD URL PROXY
				urlProxy, err := url.Parse(mod.BINDING.PROTOCOL + "://" + mod.BINDING.ADDRESS + ":" + mod.BINDING.PORT + path)
				if err != nil {
					log.Println(err) //TODO ERROR HANDLING
				}

				//TODO ADD CUSTOM HEADERS HERE

				//SETUP REVERSE PROXY DIRECTOR
				proxy := httputil.NewSingleHostReverseProxy(urlProxy)
				proxy.Director = func(req *http.Request) {

					req.URL.Scheme = urlProxy.Scheme
					req.Host = urlProxy.Host
					req.URL.Host = urlProxy.Host
					req.URL.Path = urlProxy.Path

					if _, ok := req.Header["User-Agent"]; !ok {
						req.Header.Set("User-Agent", "")
					}
				}
				proxy.ErrorHandler = ErrorHandler
				proxy.ModifyResponse = Handle404Status
				proxy.ServeHTTP(ctx.ResponseWriter, ctx.Request)
			}
		} else {
			title := ""
			code := 500
			message := ""
			if mod != nil && (mod.STATE == Loading || mod.STATE == Downloaded) {
				title = "Loading"
				code += 3
				message = "Module is loading ..."
			} else if mod != nil && mod.STATE == Stopped {
				title = "Stopped"
				code = 410
				message = "Module stopped by an administrator"
			} else if mod == nil || mod.STATE == Error || mod.STATE == Unknown {
				title = "Error"
				message = "Error"
			}

			data := ErrorPage{
				Title:   title,
				Code:    code,
				Message: message,
			}

			tmpl := template.Must(template.ParseFiles("./resources/html/loading.html"))
			tmpl.Execute(ctx.ResponseWriter, data)
		}
	})
}

// FileBind - File bind handler
func FileBind(fileName string, r Route) HandlerFunc {
	return HandlerFunc(func(ctx *Context) {
		if fileName != "" {
			http.ServeFile(ctx.ResponseWriter, ctx.Request, fileName)
		} else {
			ctx.Text(400, "GO-WOXY Core - Error Bind - "+fileName+" was not found")
		}
	})
}

// ErrorHandler -
func ErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	title := "Error"
	//message := "Error"

	data := ErrorPage{
		Title:   title,
		Code:    400,
		Message: err.Error(),
	}

	tmpl := template.Must(template.ParseFiles("./resources/html/loading.html"))
	tmpl.Execute(w, data)
}

func error404() HandlerFunc {
	return HandlerFunc(func(ctx *Context) {
		fp := path.Join("resources/html", "404.html")
		tmpl, err := template.ParseFiles(fp)
		if err != nil {
			log.Println("GO-WOXY Core - Error 404 template Not Found")
			http.Error(ctx.ResponseWriter, err.Error(), http.StatusInternalServerError)
			return
		}
		ctx.ResponseWriter.WriteHeader(404)
		if err := tmpl.Execute(ctx.ResponseWriter, nil); err != nil {
			log.Println("GO-WOXY Core - Error executing 404 template")
			http.Error(ctx.ResponseWriter, err.Error(), http.StatusInternalServerError)
		} else {
			log.Println("GO-WOXY Core - 404 Not Found")
		}
	})
}

// - Throw err when proxied response status is 404
func Handle404Status(res *http.Response) error {
	if res.StatusCode == 404 {
		return errors.New("404 error from the host")
	}
	return nil
}
