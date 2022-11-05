package main

import (
	"imagesync"
	Logger "imagesync/pkg/log"
	"os"
	"os/signal"
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
	sign := make(chan os.Signal)
	signal.Notify(sign, os.Kill, os.Interrupt)

	isync := imagesync.New(os.DirFS(root), root)

	isync.ProcessFiles()

	isync.Watch(time.Millisecond * 500)

	go func() {
		<-sign
		isync.Close()
		close(done)
	}()

	<-done

}
