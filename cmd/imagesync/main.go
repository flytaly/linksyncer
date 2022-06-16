package main

import (
	"fmt"
	"imagesync"
	"log"
	"os"
)

func main() {
	root, err := os.Getwd()

	if err != nil {
		log.Fatal(err)
	}

	isync := imagesync.New(os.DirFS(root), root)

	isync.FindFiles()
	isync.ParseFiles()

	for image, files := range isync.Images {
		fmt.Printf("%v -> %v\n", image, files)
	}

}
