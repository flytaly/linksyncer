package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/flytaly/linksyncer/pkg/fswatcher"
)

func main() {
	root, err := os.Getwd()

	if err != nil {
		log.Fatal(err)
	}

	done := make(chan struct{})

	sign := make(chan os.Signal, 1)
	signal.Notify(sign, os.Interrupt, syscall.SIGTERM)

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
			case <-watcher.ScanComplete():
				// fmt.Println("Complete scan")
			case err := <-watcher.Errors():
				log.Fatalln(err)
			}
		}
	}()

	go func() {
		err := watcher.Start(time.Millisecond * 500)
		if err != nil {
			log.Fatal(err)
		}
	}()

	go func() {
		s := <-sign
		err := watcher.Close()
		if err != nil {
			log.Fatal(err)
		}
		close(done)
		if s == syscall.SIGTERM {
			os.Exit(1)
		}
	}()

	<-done
}
