package imagesync

import (
	"fmt"
	"io/fs"
)

type ImageSync struct {
	root  string // path to the root directory
	Files []string
}

// Creates a new ImageSync
func New(root string) *ImageSync {
	return &ImageSync{
		root:  root,
		Files: []string{},
	}
}

func (s *ImageSync) FindFiles(fileSystem fs.FS, path string) {
	files, err := FileList(fileSystem, path)
	if err != nil {
		fmt.Println(err)
		return
	}

	s.Files = files
}
