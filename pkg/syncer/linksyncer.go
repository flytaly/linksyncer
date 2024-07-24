package syncer

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/flytaly/linksyncer/pkg/fswatcher"
	"github.com/flytaly/linksyncer/pkg/log"
)

type Empty struct{}

type LinkSyncer struct {
	fileSystem  fs.FS
	root        string                      // path to the root directory
	Sources     map[string][]LinkInfo       // watching files
	Linked      map[string]map[string]Empty // map linked file paths to their source files
	MaxFileSize int64                       // max file size in bytes for parsable files

	Watcher fswatcher.FsWatcher

	stopEvents chan Empty
	log        log.Logger
	mu         *sync.Mutex
}

// var watchedExt = regexp.MustCompile("(?i)(" + ImgExtensions + "|" + ParsableFilesExtension + ")$")
var parsableFiles = regexp.MustCompile("(?i)(" + ParsableFilesExtension + ")$")
var imageFiles = regexp.MustCompile("(?i)(" + ImgExtensions + ")$")

const MaxFileSize int64 = 1024 * 1024

func getShouldSkipPath(iSync *LinkSyncer) func(fs.FileInfo) bool {
	return func(fi fs.FileInfo) bool {
		name := fi.Name()
		if fi.IsDir() {
			if name == "." { // don't skip root folder
				return false
			}
			// TODO: should be optional
			return strings.HasPrefix(name, ".") || ExcludedDirs[name]
		}

		if parsableFiles.MatchString(name) {
			return fi.Size() > iSync.MaxFileSize
		}

		return !imageFiles.MatchString(name)
	}
}

// Creates a new LinkSyncer
func New(fileSystem fs.FS, root string, logger log.Logger, options ...func(*LinkSyncer)) *LinkSyncer {
	watcher := fswatcher.NewFsPoller(fileSystem, root)
	if logger == nil {
		logger = log.NewEmptyLog()
	}
	iSync := &LinkSyncer{
		root:        root,
		Watcher:     watcher,
		Sources:     map[string][]LinkInfo{},
		Linked:      map[string]map[string]Empty{},
		fileSystem:  fileSystem,
		stopEvents:  make(chan Empty),
		mu:          new(sync.Mutex),
		log:         logger,
		MaxFileSize: MaxFileSize,
	}

	for _, option := range options {
		option(iSync)
	}

	watcher.AddShouldSkipHook(getShouldSkipPath(iSync))

	return iSync
}

var extractLinks = GetLinksFromFile
var writeFile = func(absPath string, data []byte) error {
	info, err := os.Stat(absPath)
	if err != nil {
		return err
	}

	return os.WriteFile(absPath, data, info.Mode())
}

func (s *LinkSyncer) processDirs(dirs []string) {
	for _, current := range dirs {
		paths, err := s.Watcher.Add(current)
		if err != nil {
			s.log.Error("Couldn't add folder %s to watcher: %v", current, err)
			continue
		}
		for f, fi := range paths {
			if !(*fi).IsDir() && s.isParsable(f) {
				s.AddFile(f)
			}
		}
	}
}

func (s *LinkSyncer) isParsable(f string) bool {
	return parsableFiles.MatchString(f)
}

// ProcessFiles walks the file tree and adds valid files
func (s *LinkSyncer) ProcessFiles() time.Duration {
	t := time.Now()
	s.processDirs([]string{s.root})
	return time.Since(t)
}

// AddFile reads, parses and saves info about given file and its links
func (s *LinkSyncer) AddFile(relativePath string) {
	s.Sources[relativePath] = []LinkInfo{}
	data, err := s.ReadFile(relativePath)
	if err != nil {
		s.log.Error("Couldn't read file. %s", err)
		return
	}

	links, images := extractLinks(relativePath, string(data))
	s.saveLinks(relativePath, links, images)
}

func (s *LinkSyncer) AddPath(path string) {
	fi, err := fs.Stat(s.fileSystem, path)
	if err != nil {
		s.log.Error("Couldn't get FileInfo. %s", err)
		return
	}
	if !fi.IsDir() && s.isParsable(path) {
		s.AddFile(path)
		s.log.Info("Added file: %s", path)
	}
}

// clearLinkReferences deletes reference link->source in the cache
func (s *LinkSyncer) clearLinkReferences(sourceFilePath string, linkPath string) {
	delete(s.Linked[linkPath], sourceFilePath)
	if len(s.Linked[linkPath]) == 0 {
		delete(s.Linked, linkPath)
	}
}

// saveLinks updates links in the cache
func (s *LinkSyncer) saveLinks(sourceFilePath string, linkGroups ...[]LinkInfo) {
	s.Sources[sourceFilePath] = []LinkInfo{} // create new slice to clear previous links
	for _, links := range linkGroups {
		for _, link := range links {
			if s.Linked[link.rootPath] == nil {
				s.Linked[link.rootPath] = map[string]Empty{}
			}
			s.Linked[link.rootPath][sourceFilePath] = Empty{}
			s.Sources[sourceFilePath] = append(s.Sources[sourceFilePath], link)
		}
	}
}

// RemoveFile removes a file and its linked files from the cache
func (s *LinkSyncer) RemoveFile(relativePath string) {
	if linked, ok := s.Sources[relativePath]; ok {
		for _, link := range linked {
			s.clearLinkReferences(relativePath, link.rootPath)
		}
		delete(s.Sources, relativePath)
	}
}

func (s *LinkSyncer) UpdateFile(relativePath string) {
	if linked, ok := s.Sources[relativePath]; ok {
		for _, li := range linked {
			s.clearLinkReferences(relativePath, li.rootPath)
		}
		s.AddFile(relativePath)
		s.log.Info("File updated: %s", relativePath)
		return
	}
	s.AddPath(relativePath)
}

// MoveFile moves a file in the cache from `oldPath` to `newPath`
// and update links in the file's content.
// `moves` is a map of all moved files including linked files, it is used to
// correctly replace paths if source file and its links were moved simultaneously.
func (s *LinkSyncer) MoveFile(oldPath, newPath string, moves map[string]string) {
	links, ok := s.Sources[oldPath]
	if !ok {
		return
	}
	s.Sources[newPath] = s.Sources[oldPath]
	delete(s.Sources, oldPath)
	if len(links) == 0 {
		return
	}
	// collect all the links in the file,
	// if linked files were also moved, use their new location
	movedLinks := []MovedLink{}
	if moves == nil {
		moves = map[string]string{}
	}
	for _, link := range links {
		ml := MovedLink{to: moves[link.rootPath], link: link}
		if ml.to == "" { // linked file wasn't moved
			ml.to = link.rootPath
		}
		movedLinks = append(movedLinks, ml)
		s.clearLinkReferences(oldPath, link.rootPath)
	}
	s.log.Info("File moved: %s -> %s", oldPath, newPath)
	err := s.UpdateLinksInFile(newPath, movedLinks)
	if err != nil {
		s.log.Error("Couldn't update links in %s. Error: %v", newPath, err)
		return
	}
}

// UpdateLinksInFile replaces links in the file
func (s *LinkSyncer) UpdateLinksInFile(relativePath string, movedLinks []MovedLink) error {
	content, err := s.ReadFile(relativePath)
	if err != nil {
		return err
	}

	updated := ReplaceLinks(relativePath, content, movedLinks)

	err = writeFile(filepath.Join(s.root, relativePath), updated)
	if err != nil {
		return err
	}

	links, images := extractLinks(relativePath, string(updated))
	for _, link := range movedLinks {
		s.clearLinkReferences(relativePath, link.link.rootPath)
	}
	s.saveLinks(relativePath, links, images)
	s.log.Info("Links updated: %s", relativePath)

	return nil
}

// Sync receives a map of moved files (from->to) and synchronize files,
// by updating links in notes and updating cache.
func (s *LinkSyncer) Sync(moves map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// 1) At first, update the files that were moved and collect moved linked files
	movedLinks := map[string]string{}
	for from, to := range moves {
		if _, ok := s.Sources[from]; ok {
			s.MoveFile(from, to, moves)
		}
		if s.Linked[from] != nil { // if linked file was moved store it in the map
			movedLinks[from] = to
			s.log.Info("Linked file moved: %s -> %s", from, to)
		}
	}
	// 2) Then synchronize rest of the files that depends on moved linked files
	fileMap := s.getFilesToSync(movedLinks)
	for sourceFile, links := range fileMap {
		err := s.UpdateLinksInFile(sourceFile, links)
		if err != nil {
			s.log.Error("Couldn't update links in %s. Error: %v", sourceFile, err)
		}
	}
}

// getFilesToSync collects notes that should be updated due to linked files relocation
func (s *LinkSyncer) getFilesToSync(movedLinks map[string]string) map[string][]MovedLink {
	fileMap := map[string][]MovedLink{}
	for from := range movedLinks {
		files, ok := s.Linked[from]
		if !ok {
			continue
		}
		for fpath := range files {
			if _, ok := fileMap[fpath]; ok { // already added
				continue
			}
			if _, ok := s.Sources[fpath]; !ok { // file doesn't exist
				continue
			}
			for _, link := range s.Sources[fpath] { // check every link in the file
				if linkTo, ok := movedLinks[link.rootPath]; ok {
					// if link was moved, add file and its links to the map
					fileMap[fpath] = append(fileMap[fpath], MovedLink{to: linkTo, link: link})
				}
			}
		}
	}
	return fileMap
}

func (s *LinkSyncer) ReadFile(filePath string) ([]byte, error) {
	var err error
	var relativePath = filePath
	if filepath.IsAbs(relativePath) {
		_, err = filepath.Rel(s.root, filePath)

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

func (s *LinkSyncer) processEvent(event fswatcher.Event, moves *map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch event.Op {
	case fswatcher.Create:
		s.AddPath(event.Name)
	case fswatcher.Remove:
		s.RemoveFile(event.Name)
	case fswatcher.Write:
		s.UpdateFile(event.Name)
	case fswatcher.Rename:
		(*moves)[event.Name] = event.NewPath
	}
}

func (s *LinkSyncer) WatchEvents(onMoves func(moves map[string]string)) {
	moves := map[string]string{}
	for {
		select {
		case event := <-s.Watcher.Events():
			s.processEvent(event, &moves)
		case <-s.Watcher.ScanComplete():
			// TODO: scan complete event, send duration
			if len(moves) == 0 {
				break
			}
			var syncFn = s.Sync
			if onMoves != nil {
				syncFn = onMoves
			}
			syncFn(moves)
			for from := range moves {
				delete(moves, from)
			}
		case <-s.stopEvents:
			return
		case err := <-s.Watcher.Errors():
			s.log.Error("%s", err)
		}
	}
}

func (s *LinkSyncer) StartFileWatcher(interval time.Duration) {
	err := s.Watcher.Start(interval)
	if err != nil {
		s.log.Error("%s", err)
	}
}

func (s LinkSyncer) RefsNum() int {
	return len(s.Linked)
}

func (s LinkSyncer) SourcesNum() int {
	return len(s.Sources)
}

func (s *LinkSyncer) Scan() {
	s.Watcher.Scan()
}

func (s *LinkSyncer) StopFileWatcher() {
	s.Watcher.Stop()
}

func (s *LinkSyncer) StopEventListeners() {
	s.stopEvents <- Empty{}
}

func (s *LinkSyncer) Watch(interval time.Duration) {
	go s.WatchEvents(nil)
	go s.StartFileWatcher(interval)
}

func (s *LinkSyncer) Close() {
	err := s.Watcher.Close()
	if err != nil {
		fmt.Println("Couldn't close watcher")
	}
	err = s.log.Close()
	if err != nil {
		fmt.Println("Couldn't close log file")
	}
}
