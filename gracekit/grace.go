package gracekit

import (
	"context"
	"os"
	"os/signal"
	"time"
)

type GoRun func()
type PreRun func()
type Shutdown func(context.Context) error

type RunOptions struct {
	PreRun   PreRun
	GoRun    GoRun
	Shutdown Shutdown

	// TODO: 支持自定义退出信号
}

func Run(runOpts RunOptions) {

	if runOpts.PreRun != nil {
		runOpts.PreRun()
	}

	go runOpts.GoRun()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := runOpts.Shutdown(ctx); err != nil {
		panic(err)
	}
}
