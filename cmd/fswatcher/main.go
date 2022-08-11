package main

import (
	"fmt"
	"imagesync/pkg/fswatcher"
	"log"
	"os"
)

func main() {
	root, err := os.Getwd()

	if err != nil {
		log.Fatal(err)
	}

	watcher := fswatcher.NewFsPoller(os.DirFS(root))
	fmt.Println(watcher)
}
