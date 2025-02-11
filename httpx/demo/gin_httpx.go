package main

import (
	"log"

	"github.com/chhz0/go-component-base/httpx"
)

func main() {
	g := httpx.Gin()
	r := g.NewRouter()

	r.Use(LoggerMiddleware)

	r.GET("/hello/:name", func(ctx httpx.Context) error {
		name := ctx.PathParam("name")
		return ctx.JSON(200, map[string]string{"message": "Hello " + name})
	})

	r2 := r.Group("/api", LoggerMiddleware2)
	r2.GET("/hello/:name", func(ctx httpx.Context) error {
		name := ctx.PathParam("name")
		return ctx.JSON(200, map[string]string{"message": "Hello " + name})
	})

	if err := g.Serve(":8080"); err != nil {
		log.Fatal(err)
	}
}
func LoggerMiddleware(next httpx.Handler) httpx.Handler {
	return func(ctx httpx.Context) error {
		// 记录请求日志
		log.Printf("%s %s", ctx.Request().Method, ctx.Request().URL.Path)
		return next(ctx)
	}
}
func LoggerMiddleware2(next httpx.Handler) httpx.Handler {
	return func(ctx httpx.Context) error {
		// 记录请求日志
		log.Printf("this is logger2 %s %s", ctx.Request().Method, ctx.Request().URL.Path)
		return next(ctx)
	}
}

// func RecoveryMiddleware(next httpx.Handler) httpx.Handler {
// 	return func(ctx httpx.Context) error {
// 		defer func() {
// 			if err := recover(); err != nil {
// 				ctx.JSON(500, map[string]interface{}{"error": "server error"})
// 			}
// 		}()
// 		return next(ctx)
// 	}
// }
