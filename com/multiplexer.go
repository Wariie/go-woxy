package com

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

// Handler - Handler endpoint function
type HandlerFunc func(*Context)

func (h HandlerFunc) Handle(ctx *Context) {
	h(ctx)
}

// PatternRoute - Module route
type PatternRoute struct {
	Pattern     *regexp.Regexp
	Handler     HandlerFunc
	RouteConfig *RouteConfig
	Route       *Route
}

func (pr *PatternRoute) Handle(h Handler) {
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

// Router - Server router containing routes
type Router struct {
	Routes       []PatternRoute
	DefaultRoute HandlerFunc
	Middlewares  []middleware
}

// NewRouter - Init new router instance
func NewRouter(NotFoundHandler HandlerFunc) *Router {
	router := &Router{
		DefaultRoute: NotFoundHandler,
	}
	return router
}

func (r *Router) Handler(pattern string, handler Handler, routeConfig *RouteConfig, ro *Route) {
	r.Handle(pattern, handler.Handle, routeConfig, ro)
}

// Handle - Handle new router into router
func (r *Router) Handle(pattern string, handler HandlerFunc, routeConfig *RouteConfig, ro *Route) {
	re := regexp.MustCompile(pattern)
	route := PatternRoute{Pattern: re, Handler: handler, RouteConfig: routeConfig, Route: ro}
	r.Routes = append(r.Routes, route)

	//Sort routes depending on the endoint lenght
	sort.SliceStable(r.Routes, func(i, j int) bool {
		return len(r.Routes[i].Pattern.String()) > len(r.Routes[j].Pattern.String())
	})
}

// ServerHTTP - Serve route from router
func (r *Router) ServeHTTP(w http.ResponseWriter, re *http.Request) {
	ctx := &Context{Request: re, ResponseWriter: w}
	var handler Handler

	//Search route
	for _, rt := range r.Routes {
		if matches := rt.Pattern.FindStringSubmatch(ctx.URL.Path); len(matches) > 0 {

			ctx.RouteConfig = rt.RouteConfig
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
	ctx.RouteConfig = nil
	ctx.Route = nil
	r.DefaultRoute(ctx)
}

// Context -
type Context struct {
	http.ResponseWriter
	*http.Request
	Params []string
	*RouteConfig
	*Route
}

// Text - Send text to context writer
func (c *Context) Text(code int, body string) (int, error) {
	c.ResponseWriter.Header().Set("Content-Type", "text/plain")
	c.WriteHeader(code)
	return io.WriteString(c.ResponseWriter, fmt.Sprintf("%s\n", body))
}

// ByteText - Send byte text to context writer
func (c *Context) ByteText(code int, body []byte) (int, error) {
	return c.Text(code, string(body))
}

// HtmlText - Send html text to context writer
func (c *Context) HtmlText(code int, body string, data interface{}) {
	//
}

// Route - Route redirection
type Route struct {
	FROM string
	TO   string
}

// RouteConfig - Parameter to handle route redirection
type RouteConfig struct {
	NAME    string
	TYPES   string
	BINDING ServerConfig
	STATE   ModuleState
}

/*ServerConfig - Server configuration*/
type ServerConfig struct {
	ADDRESS  string
	PATH     []Route
	PORT     string
	PROTOCOL string
	ROOT     string
	CERT     string
	CERT_KEY string
}

// ModuleState - ModuleConfig State
type ModuleState int

// ModuleState list
const (
	Stopped    ModuleState = 0
	Unknown    ModuleState = 1
	Online     ModuleState = 2
	Downloaded ModuleState = 3
	Loading    ModuleState = 4

	Error  ModuleState = 999
	Failed ModuleState = 998
)
