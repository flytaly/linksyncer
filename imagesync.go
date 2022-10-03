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
	root       string                         // path to the root directory
	Files      map[string][]LinkInfo          // watching files
	Images     map[string]map[string]struct{} // map images' paths to their text files

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
		Images:     map[string]map[string]struct{}{},
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
			if !(*fi).IsDir() && s.isParsable(f) {
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

func (s *ImageSync) isParsable(f string) bool {
	return parsableFiles.MatchString(f)
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

	images := extractImages(relativePath, string(data))
	s.saveLinks(relativePath, &images)
}

func (s *ImageSync) clearLinkReferences(sourceFilePath string, linkPath string) {
	delete(s.Images[linkPath], sourceFilePath)
	if len(s.Images[linkPath]) == 0 {
		delete(s.Images, linkPath)
	}
}

func (s *ImageSync) saveLinks(sourceFilePath string, links *[]LinkInfo) {
	for _, img := range *links {
		if s.Images[img.rootPath] == nil {
			s.Images[img.rootPath] = map[string]struct{}{}
		}
		s.Images[img.rootPath][sourceFilePath] = struct{}{}
		s.Files[sourceFilePath] = append(s.Files[sourceFilePath], img)
	}
}

// Remove a file and its images from the ImageSync struct
func (s *ImageSync) RemoveFile(relativePath string) {
	if images, ok := s.Files[relativePath]; ok {
		for _, li := range images {
			s.clearLinkReferences(relativePath, li.rootPath)
		}
		delete(s.Files, relativePath)
	}
}

func (s *ImageSync) RenameFile(prevPath, newPath string) {
	if links, ok := s.Files[prevPath]; ok {
		s.Files[newPath] = s.Files[prevPath]
		delete(s.Files, prevPath)
		if len(s.Files[newPath]) == 0 {
			return
		}
		// update links
		movedLinks := []MovedLink{}
		for _, li := range links {
			movedLinks = append(movedLinks, MovedLink{
				prevPath: li.rootPath,
				newPath:  li.rootPath,
				link:     li.originalLink,
			})
			s.clearLinkReferences(prevPath, li.rootPath)
		}
		err := s.UpdateLinksInFile(newPath, movedLinks)
		if err != nil {
			fmt.Println("Couldn't update links in", newPath, err)
		}
	}
}

// UpdateLinksInFile replaces links in the file
func (s *ImageSync) UpdateLinksInFile(relativePath string, links []MovedLink) error {
	content, err := s.ReadFile(relativePath)
	if err != nil {
		return err
	}

	updated := ReplaceImageLinks(relativePath, content, links)

	err = writeFile(filepath.Join(s.root, relativePath), updated)
	if err != nil {
		return err
	}

	images := extractImages(relativePath, string(updated))
	for _, link := range links {
		s.clearLinkReferences(relativePath, link.prevPath)
	}
	s.saveLinks(relativePath, &images)

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
