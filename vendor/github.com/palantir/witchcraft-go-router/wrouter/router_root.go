// Copyright (c) 2018 Palantir Technologies. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package wrouter

import (
	"context"
	"net/http"
	"sort"

	"github.com/palantir/pkg/metrics"
)

type RootRouter interface {
	http.Handler
	Router

	AddRequestHandlerMiddleware(handlers ...RequestHandlerMiddleware)
	AddRouteHandlerMiddleware(handlers ...RouteHandlerMiddleware)

	// RegisterNotFoundHandler registers a handler to produce 404 responses.
	// It should be called after all middlewares are added to the router.
	RegisterNotFoundHandler(handler http.Handler)
}

type rootRouter struct {
	// impl stores the underlying RouterImpl used to route requests.
	impl RouterImpl

	// reqHandlers specifies the handlers that run for every request received by the router. Every request received by
	// the router (including requests to methods/paths that are not registered on the router) is handled by these
	// handlers in order before being handled by the underlying RouterImpl.
	reqHandlers []RequestHandlerMiddleware

	// routeHandlers specifies the handlers that are run for all of the routes that are registered on this router.
	// Requests that are routed to a registered route on this router are handled by these handlers in order before being
	// handled by the registered handler.
	routeHandlers []RouteHandlerMiddleware

	// routes stores all of the routes that are registered on this router.
	routes []RouteSpec

	// cachedHandler stores the http.Handler created by chaining all of the request handlers in reqHandlers with the
	// request handler provided by impl. This is done because this http.Handler is called on every request and
	// reqHandlers rarely changes, so it is much more efficient to cache the handler rather than creating a chained one
	// on every request.
	cachedHandler http.Handler
}

func New(impl RouterImpl, params ...RootRouterParam) RootRouter {
	r := &rootRouter{
		impl: impl,
	}
	for _, p := range params {
		if p == nil {
			continue
		}
		p.configure(r)
	}
	r.updateCachedHandler()

	return r
}

type RootRouterParam interface {
	configure(*rootRouter)
}

type rootRouterParamsFunc func(*rootRouter)

func (f rootRouterParamsFunc) configure(r *rootRouter) {
	f(r)
}

func RootRouterParamAddRequestHandlerMiddleware(reqHandler ...RequestHandlerMiddleware) RootRouterParam {
	return rootRouterParamsFunc(func(r *rootRouter) {
		r.AddRequestHandlerMiddleware(reqHandler...)
	})
}

func RootRouterParamAddRouteHandlerMiddleware(routeReqHandler ...RouteHandlerMiddleware) RootRouterParam {
	return rootRouterParamsFunc(func(r *rootRouter) {
		r.AddRouteHandlerMiddleware(routeReqHandler...)
	})
}

func (r *rootRouter) updateCachedHandler() {
	r.cachedHandler = createRequestHandler(r.impl, r.reqHandlers)
}

func (r *rootRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.cachedHandler.ServeHTTP(w, req)
}

func (r *rootRouter) Register(method, path string, handler http.Handler, params ...RouteParam) error {
	b := &routeParamBuilder{}
	for _, param := range params {
		if param == nil {
			continue
		}
		if err := param.apply(b); err != nil {
			return err
		}
	}

	pathTemplate, err := NewPathTemplate(path)
	if err != nil {
		return err
	}

	var pathVarNames []string
	for _, segment := range pathTemplate.Segments() {
		if segment.Type == LiteralSegment {
			continue
		}
		pathVarNames = append(pathVarNames, segment.Value)
	}

	routeSpec := RouteSpec{
		Method:       method,
		PathTemplate: pathTemplate.Template(),
	}
	r.routes = append(r.routes, routeSpec)
	sort.Sort(routeSpecs(r.routes))

	r.impl.Register(method, pathTemplate.Segments(), &routeHandler{
		impl:              r.impl,
		disableTelemetry:  b.disableTelemetry,
		metricTags:        b.toMetricTags(),
		requestParamPerms: b.toRequestParamPerms(),
		routeSpec:         routeSpec,
		pathVarNames:      pathVarNames,
		handler: func(rw http.ResponseWriter, req *http.Request, reqVals RequestVals) {
			middlewares := append(append([]RouteHandlerMiddleware{}, r.routeHandlers...), b.middleware...)
			createRouteRequestHandler(handler, middlewares)(rw, req, reqVals)
		},
	})
	return nil
}

type routeHandler struct {
	impl              RouterImpl
	disableTelemetry  bool
	metricTags        metrics.Tags
	requestParamPerms RouteParamPerms
	routeSpec         RouteSpec
	pathVarNames      []string
	handler           RouteRequestHandler
}

func (r *routeHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	pathParamVals := r.impl.PathParams(req, r.pathVarNames)
	req = req.WithContext(context.WithValue(req.Context(), pathParamsContextKey, pathParamVals))
	r.handler(w, req, RequestVals{
		Spec:             r.routeSpec,
		PathParamVals:    pathParamVals,
		ParamPerms:       r.requestParamPerms,
		MetricTags:       r.metricTags,
		DisableTelemetry: r.disableTelemetry,
	})
}

func (r *rootRouter) RegisteredRoutes() []RouteSpec {
	ris := make([]RouteSpec, len(r.routes))
	copy(ris, r.routes)
	return ris
}

func (r *rootRouter) Get(path string, handler http.Handler, params ...RouteParam) error {
	return r.Register(http.MethodGet, path, handler, params...)
}

func (r *rootRouter) Head(path string, handler http.Handler, params ...RouteParam) error {
	return r.Register(http.MethodHead, path, handler, params...)
}

func (r *rootRouter) Post(path string, handler http.Handler, params ...RouteParam) error {
	return r.Register(http.MethodPost, path, handler, params...)
}

func (r *rootRouter) Put(path string, handler http.Handler, params ...RouteParam) error {
	return r.Register(http.MethodPut, path, handler, params...)
}

func (r *rootRouter) Patch(path string, handler http.Handler, params ...RouteParam) error {
	return r.Register(http.MethodPatch, path, handler, params...)
}

func (r *rootRouter) Delete(path string, handler http.Handler, params ...RouteParam) error {
	return r.Register(http.MethodDelete, path, handler, params...)
}

func (r *rootRouter) Subrouter(path string, params ...RouteParam) Router {
	return &subrouter{
		rPath:   path,
		rParent: r,
		params:  params,
	}
}

func (r *rootRouter) Path() string {
	return ""
}

func (r *rootRouter) Parent() Router {
	return nil
}

func (r *rootRouter) RootRouter() RootRouter {
	return r
}

func (r *rootRouter) AddRequestHandlerMiddleware(handlers ...RequestHandlerMiddleware) {
	r.reqHandlers = append(r.reqHandlers, handlers...)
	r.updateCachedHandler()
}

func (r *rootRouter) AddRouteHandlerMiddleware(handlers ...RouteHandlerMiddleware) {
	r.routeHandlers = append(r.routeHandlers, handlers...)
}

func (r *rootRouter) RegisterNotFoundHandler(handler http.Handler) {
	r.impl.RegisterNotFoundHandler(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		createRouteRequestHandler(handler, r.routeHandlers)(rw, req, RequestVals{
			Spec: RouteSpec{
				Method:       req.Method,
				PathTemplate: "/*",
			},
			PathParamVals: map[string]string{},
			ParamPerms: &requestParamPermsImpl{
				pathParamPerms:   newParamPerms(nil, nil),
				queryParamPerms:  newParamPerms(nil, nil),
				headerParamPerms: newParamPerms(nil, nil),
			},
			MetricTags: metrics.Tags{},
		})
	}))
}

type requestHandlerWithNext struct {
	handler RequestHandlerMiddleware
	next    http.Handler
}

func createRequestHandler(baseHandler http.Handler, handlers []RequestHandlerMiddleware) http.Handler {
	if len(handlers) == 0 {
		return baseHandler
	}
	return &requestHandlerWithNext{
		handler: handlers[0],
		next:    createRequestHandler(baseHandler, handlers[1:]),
	}
}

func (r *requestHandlerWithNext) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	r.handler(rw, req, r.next)
}

func createRouteRequestHandler(baseHandler http.Handler, handlers []RouteHandlerMiddleware) RouteRequestHandler {
	if len(handlers) == 0 {
		return func(rw http.ResponseWriter, r *http.Request, _ RequestVals) { baseHandler.ServeHTTP(rw, r) }
	}
	return func(rw http.ResponseWriter, req *http.Request, reqVals RequestVals) {
		handlers[0](rw, req, reqVals, createRouteRequestHandler(baseHandler, handlers[1:]))
	}
}
