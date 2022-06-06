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

	isync := imagesync.New(root)

	fsystem := os.DirFS(root)
	isync.FindFiles(fsystem, ".")

	files, _ := imagesync.FileList(os.DirFS(root), ".")

	for _, v := range files {
		images, err := imagesync.GetImagesFromFile(os.DirFS(root), v)

		if err != nil {
			fmt.Println(err)
			continue
		}

		fmt.Printf("%v images %v\n", v, images)
	}
}
