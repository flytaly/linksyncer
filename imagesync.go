package imagesync

import (
	"io/fs"
	"log"
	"os"
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
var writeFile = func(filePath string, data []byte) error {
	return os.WriteFile(filePath, data, 0644)
}

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
	data, err := s.ReadFile(filePath)
	if err != nil {
		log.Println(err)
		return
	}
	s.ParseFile(filePath, string(data))
}

// Extract image paths from supported files and add them into `Images`
func (s *ImageSync) ParseFile(filePath, fileContent string) {
	images := extractImages(filePath, fileContent)

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
			if len(s.Images[image.absPath]) == 0 {
				delete(s.Images, image.absPath)
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

func (s *ImageSync) UpdateImageLinks(filePath string, images []RenamedImage) error {
	file, err := s.ReadFile(filePath)

	if err != nil {
		return err
	}

	updated := ReplaceImageLinks(filePath, file, images)

	err = writeFile(filePath, updated)

	if err != nil {
		return err
	}

	s.RemoveFile(filePath)
	s.ParseFile(filePath, string(updated))

	return err
}

func (s *ImageSync) ReadFile(filePath string) ([]byte, error) {
	relativePath, err := filepath.Rel(s.root, filePath)

	if err != nil {
		return nil, err
	}

	data, err := fs.ReadFile(s.fileSystem, relativePath)

	if err != nil {
		return nil, err
	}

	return data, nil
}

func filter(ss []string, test func(string) bool) (res []string) {
	for _, s := range ss {
		if test(s) {
			res = append(res, s)
		}
	}
	return
}
