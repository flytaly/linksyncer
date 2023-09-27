package main

import (
	"imagesync"
	Logger "imagesync/pkg/log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	root, err := os.Getwd()

	log := Logger.New()

	if err != nil {
		log.Error("%s", err)
		os.Exit(1)
	}

	done := make(chan struct{})
	sign := make(chan os.Signal, 1)
	signal.Notify(sign, os.Interrupt, syscall.SIGTERM)

	isync := imagesync.New(os.DirFS(root), root)

	isync.ProcessFiles()

	isync.Watch(time.Millisecond * 500)

	go func() {
		s := <-sign
		isync.Close()
		close(done)
		if s == syscall.SIGTERM {
			os.Exit(1)
		}
	}()

	<-done
}
