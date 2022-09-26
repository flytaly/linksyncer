package main

import (
	"imagesync/pkg/fswatcher"
	"log"
	"os"
	"os/signal"
	"path/filepath"
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

	watcher := fswatcher.NewFsPoller(os.DirFS(root), root)

	err = watcher.Add(filepath.Join(root, "." /* "test_files" */))

	if err != nil {
		log.Fatal(err)
	}

	go watcher.Start(time.Second * 1)

	go func() {
		<-sign
		watcher.Close()
		close(done)
	}()

	<-done
}