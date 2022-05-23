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

	for _, v := range files {
		images, err := imagesync.GetImagesFromFile(os.DirFS(root), v)

		if err != nil {
			fmt.Println(err)
			continue
		}

		fmt.Printf("%v images %v\n", v, images)
	}

}
