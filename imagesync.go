package imagesync

import (
	"fmt"
	"imagesync/pkg/fswatcher"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

type ImageSync struct {
	fileSystem fs.FS
	root       string                // path to the root directory
	Files      map[string][]LinkInfo // watching files
	Images     map[string][]string   // map images' paths to their text files

	watcher fswatcher.FsWatcher

	mu *sync.Mutex
}

var parsableFiles = regexp.MustCompile("(?i)(" + ParsableFilesExtension + ")$")

var watchedExt = regexp.MustCompile("(?i)(" + ImgExtensions + "|" + ParsableFilesExtension + ")$")

func shouldSkipPath(fi fs.FileInfo) bool {
	name := fi.Name()
	if fi.IsDir() {
		// TODO: should be optional
		return strings.HasPrefix(name, ".") || ExcludedDirs[name]
	}

	return !watchedExt.MatchString(name)
}

// Creates a new ImageSync
func New(fileSystem fs.FS, root string) *ImageSync {
	watcher := fswatcher.NewFsPoller(fileSystem, root)
	watcher.AddShouldSkipHook(shouldSkipPath)
	return &ImageSync{
		root:       root,
		watcher:    watcher,
		Files:      map[string][]LinkInfo{},
		Images:     map[string][]string{},
		fileSystem: fileSystem,
		mu:         new(sync.Mutex),
	}
}

var extractImages = GetImagesFromFile
var writeFile = func(absPath string, data []byte) error {
	return os.WriteFile(absPath, data, 0644)
}

func (s *ImageSync) processDirs(dirs []string) {
	nestedDirs := []string{}
	for _, current := range dirs {
		paths, err := s.watcher.Add(current)
		if err != nil {
			fmt.Printf("Couldn't add folder %s to watcher: %v", current, err)
			continue
		}
		for f, fi := range paths {
			if !(*fi).IsDir() && parsableFiles.MatchString(f) {
				s.AddFile(f)
				continue
			}
			if f != "." && f != current {
				nestedDirs = append(nestedDirs, f)
			}
		}
	}
	if len(nestedDirs) > 0 {
		s.processDirs(nestedDirs)
	}
}

// Walks the file tree and fill Images and Files maps
func (s *ImageSync) ProcessFiles() {
	s.processDirs([]string{s.root})
}
func (s *ImageSync) AddFile(relativePath string) {
	s.Files[relativePath] = []LinkInfo{}
	data, err := s.ReadFile(relativePath)
	if err != nil {
		log.Println(err)
		return
	}
	s.ParseFileContent(relativePath, string(data))
}

// Extract image paths from supported files and add them into `Images`
func (s *ImageSync) ParseFileContent(relativePath, fileContent string) {
	images := extractImages(relativePath, fileContent)

	for _, img := range images {
		s.Images[img.rootPath] = append(s.Images[img.rootPath], relativePath)
		s.Files[relativePath] = append(s.Files[relativePath], img)
	}
}

// Remove a file and its images from the ImageSync struct
func (s *ImageSync) RemoveFile(relativePath string) {
	if images, ok := s.Files[relativePath]; ok {
		for _, image := range images {
			if files, ok := s.Images[image.rootPath]; ok {
				s.Images[image.rootPath] = filter(files, func(s string) bool { return s != relativePath })
			}
			if len(s.Images[image.rootPath]) == 0 {
				delete(s.Images, image.rootPath)
			}

		}
		delete(s.Files, relativePath)
	}
}

func (s *ImageSync) RenameFile(prevPath, newPath string) {
	// TODO: Image links in the file should be updated after file relocation

	s.RemoveFile(prevPath)
	s.AddFile(newPath)
}

// UpdateLinksInFile replaces links in the file
func (s *ImageSync) UpdateLinksInFile(relativePath string, links []RenamedImage) error {
	content, err := s.ReadFile(relativePath)
	if err != nil {
		return err
	}

	updated := ReplaceImageLinks(relativePath, content, links)

	err = writeFile(filepath.Join(s.root, relativePath), updated)
	if err != nil {
		return err
	}

	s.RemoveFile(relativePath)
	s.ParseFileContent(relativePath, string(updated))

	return nil
}

func (s *ImageSync) ReadFile(filePath string) ([]byte, error) {
	var err error
	var relativePath = filePath
	if filepath.IsAbs(relativePath) {
		relativePath, err = filepath.Rel(s.root, filePath)

		if err != nil {
			return nil, err
		}
	}

	data, err := fs.ReadFile(s.fileSystem, filePath)

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
