package imagesync

import (
	"io/fs"
	"path/filepath"
	"strings"
)

var excludedDirs = map[string]bool{"node_modules": true}

func shouldSkip(name string) bool {
	if name == "." {
		return false
	}

	return strings.HasPrefix(name, ".") || excludedDirs[name]
}

func FileList(fileSystem fs.FS, path string) ([]string, error) {
	var files []string

	err := fs.WalkDir(fileSystem, path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// skip hidden and some other dirs
		if d.IsDir() && shouldSkip(d.Name()) {
			return filepath.SkipDir
		}

		if d.IsDir() || d.Name() == "." {
			return nil
		}

		files = append(files, path)
		return nil
	})

	return files, err
}
