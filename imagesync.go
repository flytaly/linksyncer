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
	for _, current := range dirs {
		paths, err := s.watcher.Add(current)
		if err != nil {
			fmt.Printf("Couldn't add folder %s to watcher: %v", current, err)
			continue
		}
		for f, fi := range paths {
			if !(*fi).IsDir() && s.isParsable(f) {
				s.AddFile(f)
			}
		}
	}
}

func (s *ImageSync) isParsable(f string) bool {
	return parsableFiles.MatchString(f)
}

// ProcessFiles walks the file tree and adds valid files
func (s *ImageSync) ProcessFiles() {
	s.processDirs([]string{s.root})
}

// AddFile reads,  parses and save info about given file and its links
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

// clearLinkReferences deletes reference link->source in the cache
func (s *ImageSync) clearLinkReferences(sourceFilePath string, linkPath string) {
	delete(s.Images[linkPath], sourceFilePath)
	if len(s.Images[linkPath]) == 0 {
		delete(s.Images, linkPath)
	}
}

// saveLinks updates links in the cache
func (s *ImageSync) saveLinks(sourceFilePath string, links *[]LinkInfo) {
	s.Files[sourceFilePath] = []LinkInfo{} // create new slice to clear previous links
	for _, img := range *links {
		if s.Images[img.rootPath] == nil {
			s.Images[img.rootPath] = map[string]struct{}{}
		}
		s.Images[img.rootPath][sourceFilePath] = struct{}{}
		s.Files[sourceFilePath] = append(s.Files[sourceFilePath], img)
	}
}

// RemoveFile removes a file and its images from the cache
func (s *ImageSync) RemoveFile(relativePath string) {
	if images, ok := s.Files[relativePath]; ok {
		for _, li := range images {
			s.clearLinkReferences(relativePath, li.rootPath)
		}
		delete(s.Files, relativePath)
	}
}

// MoveFile moves a file in the cache from `oldPath` to `newPath`
// and update links in the file's content.
// `moves` is a map of all moved files including linked files, it is used to
// correctly replace paths if source file and its links were moved simultaneously.
func (s *ImageSync) MoveFile(oldPath, newPath string, moves map[string]string) {
	links, ok := s.Files[oldPath]
	if !ok {
		return
	}
	s.Files[newPath] = s.Files[oldPath]
	delete(s.Files, oldPath)
	if len(links) == 0 {
		return
	}
	// collect all the links in the file,
	// in linked files were also moved, use their new location
	movedLinks := []MovedLink{}
	if moves == nil {
		moves = map[string]string{}
	}
	for _, li := range links {
		ml := MovedLink{from: li.rootPath, to: moves[li.rootPath], link: li.originalLink}
		if ml.to == "" { // linked file wasn't moved
			ml.to = li.rootPath
		}
		movedLinks = append(movedLinks, ml)
		s.clearLinkReferences(oldPath, li.rootPath)
	}
	err := s.UpdateLinksInFile(newPath, movedLinks)
	if err != nil {
		fmt.Printf("Couldn't update links in %s. Error: %v", newPath, err)
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
		s.clearLinkReferences(relativePath, link.from)
	}
	s.saveLinks(relativePath, &images)

	return nil
}

// Sync receives a map of moved files (from->to) and synchronize files,
// by updating links in notes and updating cache.
func (s *ImageSync) Sync(moves map[string]string) {
	// 1) At first, update the files that were moved and collect moved linked files
	movedLinks := map[string]string{}
	for from, to := range moves {
		if _, ok := s.Files[from]; ok {
			s.MoveFile(from, to, moves)
		}
		if s.Images[from] != nil { // if linked file was moved store it in the map
			movedLinks[from] = to
		}
	}
	// 2) Then synchronize rest of the files that depends on moved linked files
	fileMap := s.getFilesToSync(movedLinks)
	for sourceFile, links := range fileMap {
		err := s.UpdateLinksInFile(sourceFile, links)
		if err != nil {
			fmt.Printf("Couldn't update links in %s. Error: %v", sourceFile, err)
		}
	}
}

// getFilesToSync collects notes that should be updated due to linked files relocation
func (s *ImageSync) getFilesToSync(movedLinks map[string]string) map[string][]MovedLink {
	fileMap := map[string][]MovedLink{}
	for from := range movedLinks {
		files, ok := s.Images[from]
		if !ok {
			continue
		}
		for fpath := range files {
			if _, ok := fileMap[fpath]; ok { // already added
				continue
			}
			if _, ok := s.Files[fpath]; !ok { // file doesn't exist
				continue
			}
			for _, li := range s.Files[fpath] { // check every link in the file
				if linkTo, ok := movedLinks[li.rootPath]; ok {
					// if link was moved, add file and its links to the map
					fileMap[fpath] = append(fileMap[fpath], MovedLink{
						from: li.rootPath, to: linkTo, link: li.originalLink,
					})
				}
			}
		}
	}
	return fileMap
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
