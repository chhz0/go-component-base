package xhttp

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type shutdownFunc func(context.Context) error

type GracefulShutdown struct {
	shutdownFuncs   []shutdownFunc
	shutdownTimeout time.Duration
	signals         []os.Signal
}

func NewGracefulShutdown() *GracefulShutdown {
	return &GracefulShutdown{
		shutdownTimeout: time.Second * 15,
		signals:         []os.Signal{os.Interrupt, syscall.SIGTERM},
	}
}
func (g *GracefulShutdown) AddShutdownFunc(shutdownFunc shutdownFunc) {
	g.shutdownFuncs = append(g.shutdownFuncs, shutdownFunc)
}

func (g *GracefulShutdown) SetTimeout(timeout time.Duration) {
	g.shutdownTimeout = timeout
}

func (g *GracefulShutdown) AddSingnal(sigs ...os.Signal) {
	g.signals = append(g.signals, sigs...)
}

func (g *GracefulShutdown) Wait() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, g.signals...)
	<-quit

	ctx, cannel := context.WithTimeout(context.Background(), g.shutdownTimeout)
	defer cannel()

	for _, shutdownFunc := range g.shutdownFuncs {
		if err := shutdownFunc(ctx); err != nil {
			panic(err)
		}
	}
}
