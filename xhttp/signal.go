package xhttp

import (
	"os"
	"os/signal"
	"syscall"
)

var (
	onlyOneSignalHandler = make(chan struct{})
	shutdownHandler      chan os.Signal
	shutdownSignals      = []os.Signal{os.Interrupt, syscall.SIGTERM}
)

func SetSignalHandler(signals []os.Signal) chan struct{} {
	close(onlyOneSignalHandler)

	signal.Notify(shutdownHandler, shutdownSignals...)
	if len(signals) > 0 {
		signal.Notify(shutdownHandler, signals...)
	}

	quit := make(chan struct{})

	go func() {
		<-shutdownHandler
		close(quit)
		<-shutdownHandler
		os.Exit(1)
	}()

	return quit
}
