package main

import (
	"fmt"
	imagesync "imagesync/src"
	"log"
	"os"
)

func main() {
	root, err := os.Getwd()

	if err != nil {
		log.Println(err)
	}

	files, _ := imagesync.FileList(os.DirFS(root), ".")

	fmt.Println(files)
}
