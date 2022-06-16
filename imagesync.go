package imagesync

import (
	"fmt"
	"io/fs"
	"log"
)

type ImageSync struct {
	fileSystem fs.FS
	root       string              // path to the root directory
	Dirs       PathList            // watching directories
	Files      PathList            // watching files
	Images     map[string][]string // map images' paths to their text files
}

// Creates a new ImageSync
func New(fileSystem fs.FS, root string) *ImageSync {
	return &ImageSync{
		root:       root,
		Dirs:       PathList{},
		Files:      PathList{},
		Images:     map[string][]string{},
		fileSystem: fileSystem,
	}
}

func (s *ImageSync) FindFiles() {
	files, err := WatchList(s.fileSystem, ".")
	if err != nil {
		log.Println(err)
		return
	}

	for path, fi := range files {
		if fi.IsDir() {
			s.Dirs[path] = fi
		} else {
			s.Files[path] = fi
		}
	}

}

func (s *ImageSync) ParseFiles() {
	for filePath := range s.Files {
		images, err := GetImagesFromFile(s.fileSystem, filePath)

		if err != nil {
			fmt.Println(err)
			continue
		}

		for _, v := range images {
			s.Images[v] = append(s.Images[v], filePath)
		}
	}
}
