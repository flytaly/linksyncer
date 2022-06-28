package imagesync

import (
	"io/fs"
	"log"
	"path/filepath"
)

type ImageSync struct {
	fileSystem fs.FS
	root       string                 // path to the root directory
	Dirs       map[string]bool        // watching directories
	Files      map[string][]ImageInfo // watching files
	Images     map[string][]string    // map images' paths to their text files
}

// Creates a new ImageSync
func New(fileSystem fs.FS, root string) *ImageSync {
	return &ImageSync{
		root:       root,
		Dirs:       map[string]bool{},
		Files:      map[string][]ImageInfo{},
		Images:     map[string][]string{},
		fileSystem: fileSystem,
	}
}

var watchList = WatchList
var extractImages = GetImagesFromFile

// Walks the file tree and fill Images and Files maps
func (s *ImageSync) ProcessFiles() {
	dirs, files, err := watchList(s.fileSystem, s.root)
	if err != nil {
		log.Println(err)
		return
	}

	for _, path := range dirs {
		s.Dirs[path] = true
	}
	for _, path := range files {
		s.AddFile(path)
	}
}

func (s *ImageSync) AddFile(filePath string) {
	s.Files[filePath] = []ImageInfo{}
	s.ParseFile(filePath)
}

// Extract image paths from supported files and add them into `Images`
func (s *ImageSync) ParseFile(filePath string) {
	relativePath, err := filepath.Rel(s.root, filePath)

	if err != nil {
		log.Println(err)
		return
	}

	images, err := extractImages(s.fileSystem, relativePath, s.root)

	if err != nil {
		log.Println(err)
		return
	}

	for _, img := range images {
		s.Images[img.absPath] = append(s.Images[img.absPath], filePath)
		s.Files[filePath] = append(s.Files[filePath], img)
	}
}

// Remove a file and its images from the ImageSync struct
func (s *ImageSync) RemoveFile(filePath string) {
	if images, ok := s.Files[filePath]; ok {
		for _, image := range images {
			if files, ok := s.Images[image.absPath]; ok {
				s.Images[image.absPath] = filter(files, func(s string) bool { return s != filePath })
			}

		}
		delete(s.Files, filePath)
	}
}

func (s *ImageSync) RenameFile(prevPath, newPath string) {
	// TODO: Image links in the file should be updated after file relocation

	s.RemoveFile(prevPath)
	s.AddFile(newPath)
}

func filter(ss []string, test func(string) bool) (res []string) {
	for _, s := range ss {
		if test(s) {
			res = append(res, s)
		}
	}
	return
}
