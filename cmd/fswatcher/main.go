package main

import (
	"imagesync/pkg/fswatcher"
	"log"
	"os"
	"path/filepath"
	"time"
)

func main() {
	root, err := os.Getwd()

	if err != nil {
		log.Fatal(err)
	}
	var done chan bool
	watcher := fswatcher.NewFsPoller(os.DirFS(root), time.Millisecond*300)

	watcher.Add(filepath.Join(root, "test_files"))

	<-done
}
