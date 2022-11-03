package main

import (
	"imagesync"
	"log"
	"os"
	"os/signal"
	"time"
)

func main() {
	root, err := os.Getwd()

	if err != nil {
		log.Fatal(err)
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
