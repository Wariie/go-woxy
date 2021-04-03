package core

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
)

type Handler interface {
	Handle(ctx *Context)
}

//Handler - Handler endpoint function
type HandlerFunc func(*Context)

func (h HandlerFunc) Handle(ctx *Context) {
	h(ctx)
}

//PatternRoute - Module route
type PatternRoute struct {
	Pattern *regexp.Regexp
	Handler HandlerFunc
	Module  *ModuleConfig
	Route   *Route
}

func (pr PatternRoute) Handle(h Handler) {
	pr.Handler = HandlerFunc(h.Handle)
}

type MiddlewareFunc func(Handler) Handler

// Middleware allows MiddlewareFunc to implement the middleware interface.
func (mw MiddlewareFunc) Middleware(handler Handler) Handler {
	return mw(handler)
}

type middleware interface {
	Middleware(handler Handler) Handler
}

//Router - Server router containing routes
type Router struct {
	Routes       []PatternRoute
	DefaultRoute HandlerFunc
	Middlewares  []middleware
}

//NewRouter - Init new router instance
func NewRouter(NotFoundHandler HandlerFunc) *Router {
	router := &Router{
		DefaultRoute: NotFoundHandler,
	}
	return router
}

type HandleFunc interface {
	Handle(h *Context)
}

func (r *Router) Handler(pattern string, handler Handler, mod *ModuleConfig, ro *Route) {
	r.Handle(pattern, handler.Handle, mod, ro)
}

//Handle - Handle new router into router
func (r *Router) Handle(pattern string, handler HandlerFunc, mod *ModuleConfig, ro *Route) {
	re := regexp.MustCompile(pattern)
	route := PatternRoute{Pattern: re, Handler: handler, Module: mod, Route: ro}
	r.Routes = append(r.Routes, route)

	//Sort routes depending on the endoint lenght
	sort.SliceStable(r.Routes, func(i, j int) bool {
		return len(r.Routes[i].Pattern.String()) > len(r.Routes[j].Pattern.String())
	})
}

//ServerHTTP - Serve route from router
func (r *Router) ServeHTTP(w http.ResponseWriter, re *http.Request) {
	ctx := &Context{Request: re, ResponseWriter: w}
	var handler Handler

	//Search route
	for _, rt := range r.Routes {
		if matches := rt.Pattern.FindStringSubmatch(ctx.URL.Path); len(matches) > 0 {

			ctx.ModuleConfig = rt.Module
			ctx.Route = rt.Route

			if len(matches) > 1 {
				ctx.Params = matches[1:]
			}

			handler = rt.Handler
			break
		}
	}

	if handler != nil {
		//Process middlewares
		for _, mw := range r.Middlewares {
			handler = mw.Middleware(handler)
		}

		//If handler redirect request
		handler.Handle(ctx)
		return
	}

	//Else route to NotFound page
	ctx.ModuleConfig = nil
	ctx.Route = nil
	r.DefaultRoute(ctx)
}

//Context -
type Context struct {
	http.ResponseWriter
	*http.Request
	Params []string
	*ModuleConfig
	*Route
}

//Text - Send text to context writer
func (c *Context) Text(code int, body string) (int, error) {
	c.ResponseWriter.Header().Set("Content-Type", "text/plain")
	c.WriteHeader(code)
	return io.WriteString(c.ResponseWriter, fmt.Sprintf("%s\n", body))
}

//ByteText - Send byte text to context writer
func (c *Context) ByteText(code int, body []byte) (int, error) {
	return c.Text(code, string(body))
}

//HtmlText - Send html text to context writer
func (c *Context) HtmlText(code int, body string, data interface{}) {
	//
}
