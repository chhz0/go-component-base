package xhttp

import (
	"context"
	"errors"
	"net/http"
	"os"
	"syscall"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
)

type GinServer struct {
	*gin.Engine

	Config *Config

	httpServer *http.Server
	tlsServer  *http.Server
}

func Gin() *GinServer {
	conf := NewZeroConfig()

	g := &GinServer{
		Config: conf,
	}

	return g
}

func (g *GinServer) Router(router func(ge *gin.Engine)) {
	router(g.Engine)
}

func (g *GinServer) Run() error {
	g.httpServer = &http.Server{
		Addr:    g.Config.Http.Address(),
		Handler: g,
	}

	g.tlsServer = &http.Server{
		Addr:    g.Config.TLS.Address(),
		Handler: g,
	}

	var eg errgroup.Group

	eg.Go(func() error {
		if err := g.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})

	eg.Go(func() error {
		cert, key := g.Config.TLS.CertKey.Cert, g.Config.TLS.CertKey.Key
		if err := g.tlsServer.ListenAndServeTLS(cert, key); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})

	// ctx, cancel := context.WithTimeout(context.Background(), g.Config.ShutdownTimeout)
	// defer cancel()

	// if err := g.ping(ctx); err != nil {
	// 	return err
	// }

	if err := eg.Wait(); err != nil {
		return err
	}

	return nil
}

func (g *GinServer) Shutdown() error {
	if err := g.httpServer.Shutdown(context.Background()); err != nil {
		return err
	}

	if err := g.tlsServer.Shutdown(context.Background()); err != nil {
		return err
	}

	return nil
}

func (g *GinServer) GracefulShutdown() {
	gs := GracefulShutdown{
		shutdownTimeout: g.Config.ShutdownTimeout,
		signals:         []os.Signal{syscall.SIGINT, syscall.SIGTERM},
		shutdownFuncs: []shutdownFunc{
			g.httpServer.Shutdown,
			g.tlsServer.Shutdown,
		},
	}

	gs.Wait()
}

var GinMiddlewares = defaultGinMiddlewares()

func defaultGinMiddlewares() map[string]gin.HandlerFunc {
	return map[string]gin.HandlerFunc{}
}
func (g *GinServer) UseMiddlewares(middlewares ...gin.HandlerFunc) {
	g.Use(middlewares...)
}

// TODO: add ping for health check
func (g *GinServer) ping(ctx context.Context) error {
	return nil
}
