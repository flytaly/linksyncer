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

	watcher := fswatcher.NewFsPoller(os.DirFS(root))

	watcher.Add(filepath.Join(root, "test_files"))

	go watcher.Start(time.Millisecond * 500)

	go func() {
		<-sign
		watcher.Close()
		close(done)
	}()

	<-done
}
