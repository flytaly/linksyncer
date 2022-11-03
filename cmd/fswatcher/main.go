package main

import (
	"fmt"
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

	_, err = watcher.Add(filepath.Join(root, "/test_files"))

	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			select {
			case event := <-watcher.Events():
				fmt.Printf("Event: %s, Path: %s, NewPath: %s\n", event.Op, event.Name, event.NewPath) // Print the event's info.
			case err := <-watcher.Errors():
				log.Fatalln(err)
			}
		}
	}()

	go watcher.Start(time.Millisecond * 500)

	go func() {
		<-sign
		watcher.Close()
		close(done)
	}()

	<-done
}
