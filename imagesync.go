package imagesync

import (
	"fmt"
	"io/fs"
	"log"
)

type ImageSync struct {
	fileSystem fs.FS
	root       string              // path to the root directory
	Dirs       map[string]bool     // watching directories
	Files      map[string][]string // watching files
	Images     map[string][]string // map images' paths to their text files
}

// Creates a new ImageSync
func New(fileSystem fs.FS, root string) *ImageSync {
	return &ImageSync{
		root:       root,
		Dirs:       map[string]bool{},
		Files:      map[string][]string{},
		Images:     map[string][]string{},
		fileSystem: fileSystem,
	}
}

var watchList = WatchList
var extractImages = GetImagesFromFile

func (s *ImageSync) ProcessFiles() {
	dirs, files, err := watchList(s.fileSystem, ".")
	if err != nil {
		log.Println(err)
		return
	}

	for _, path := range dirs {
		s.Dirs[path] = true
	}
	for _, path := range files {
		s.Files[path] = []string{}
		s.ParseFile(path)
	}
}

func (s *ImageSync) ParseFile(filePath string) {
	images, err := extractImages(s.fileSystem, filePath)

	if err != nil {
		fmt.Println(err)
		return
	}

	for _, v := range images {
		s.Images[v] = append(s.Images[v], filePath)
		s.Files[filePath] = append(s.Files[filePath], v)
	}
}
