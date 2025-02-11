package httpx

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Gin() Adapter {
	adapter = NewGinAdapter()
	return adapter
}

type GinAdapter struct {
	server *http.Server
	engine *gin.Engine
}

func NewGinAdapter() Adapter {
	engine := gin.New()
	return &GinAdapter{
		engine: engine,
	}
}

// NewRouter implements Adapter.
func (ga *GinAdapter) NewRouter() Router {
	return &ginRouter{
		engine: ga.engine,
	}
}

// ConverHandler implements Adapter. 转换 httpx.Handler 为 gin.HandlerFunc
func (ga *GinAdapter) ConverHandler(h Handler) interface{} {
	return func(c *gin.Context) {
		// TODO: 对象池获取上下文
		ctx := getContext()
		defer releaseContext(ctx)

		// 设置上下文
		ctx.reset()
		ctx.request = c.Request
		ctx.response = c.Writer
		ctx.params = ginParamsToMap(c.Params)

		if err := h(ctx); err != nil {
			_ = c.Error(err)
		}
	}
}

// WrapContext implements Adapter. 包装 Gin Context
func (ga *GinAdapter) WrapContext(ctx interface{}) Context {
	c := ctx.(*gin.Context)
	return &baseContext{
		request:  c.Request,
		response: c.Writer,
		params:   ginParamsToMap(c.Params),
	}
}

// func (ga *GinAdapter) WrapMiddleware(m Middleware) gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		next := func(ctx Context) error {
// 			c.Next()
// 			return nil
// 		}
// 		handler := m(next)
// 		_ = handler(ga.WrapContext(c))
// 	}
// }

// Serve implements Adapter. 启动服务
func (ga *GinAdapter) Serve(addr string) error {
	ga.server = &http.Server{
		Addr:    addr,
		Handler: ga.engine,
	}
	return ga.server.ListenAndServe()
}

// Shutdown implements Adapter. 关闭服务
func (ga *GinAdapter) Shutdown(ctx context.Context) error {
	if ga.server != nil {
		return ga.server.Shutdown(ctx)
	}
	return nil
}

// ginRouter 封装gin路由器
type ginRouter struct {
	engine *gin.Engine
	group  *gin.RouterGroup
}

// GET implements Router.
func (r *ginRouter) GET(path string, h Handler, ms ...Middleware) {
	r.handle(http.MethodGet, path, h, ms...)
}

// POST implements Router.
func (r *ginRouter) POST(path string, h Handler, ms ...Middleware) {
	r.handle(http.MethodPost, path, h, ms...)
}

// PUT implements Router.
func (r *ginRouter) PUT(path string, h Handler, ms ...Middleware) {
	r.handle(http.MethodPut, path, h, ms...)
}

// DELETE implements Router.
func (r *ginRouter) DELETE(path string, h Handler, ms ...Middleware) {
	r.handle(http.MethodDelete, path, h, ms...)
}

// PATCH implements Router.
func (r *ginRouter) PATCH(path string, h Handler, ms ...Middleware) {
	r.handle(http.MethodPatch, path, h, ms...)
}

// HEAD implements Router.
func (r *ginRouter) HEAD(path string, h Handler, ms ...Middleware) {
	r.handle(http.MethodHead, path, h, ms...)
}

// OPTIONS implements Router.
func (r *ginRouter) OPTIONS(path string, h Handler, ms ...Middleware) {
	r.handle(http.MethodOptions, path, h, ms...)
}

// Static implements Router.
func (r *ginRouter) Static(prefix string, root string) {
	if r.group != nil {
		r.group.Static(prefix, root)
	} else {
		r.engine.Static(prefix, root)
	}
}

// Group implements Router.
func (r *ginRouter) Group(prefix string, ms ...Middleware) Router {
	var ginMiddlewares []gin.HandlerFunc
	for _, middleware := range ms {
		ginMiddlewares = append(ginMiddlewares, wrapMiddleware(middleware))
	}

	return &ginRouter{
		engine: r.engine,
		group:  r.engine.Group(prefix, ginMiddlewares...),
	}
}

// Use implements Router.
func (r *ginRouter) Use(ms ...Middleware) {
	for _, m := range ms {
		r.engine.Use(wrapMiddleware(m))
	}
}

func (r *ginRouter) handle(method, path string, h Handler, ms ...Middleware) {
	handlerChain := buildHandlerChain(h, ms...)
	ginHandler := handlerChain.(gin.HandlerFunc)

	if r.group != nil {
		r.group.Handle(method, path, ginHandler)
	} else {
		r.engine.Handle(method, path, ginHandler)
	}
}

// 构建处理链
func buildHandlerChain(h Handler, ms ...Middleware) interface{} {
	handler := h
	// 反向应用中间件
	for i := len(ms) - 1; i >= 0; i-- {
		handler = ms[i](handler)
	}
	return gin.HandlerFunc(func(ctx *gin.Context) {
		_ = handler(adapter.WrapContext(ctx))
	})
}

// 中间件转换
func wrapMiddleware(m Middleware) gin.HandlerFunc {
	return func(c *gin.Context) {
		next := func(ctx Context) error {
			c.Next()
			return nil
		}
		handler := m(next)
		_ = handler(adapter.WrapContext(c))
	}
}
func ginParamsToMap(params gin.Params) map[string]string {
	result := make(map[string]string)
	for _, param := range params {
		result[param.Key] = param.Value
	}
	return result
}
