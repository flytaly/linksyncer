package imagesync

import (
	"errors"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
)

var excludedDirs = map[string]bool{"node_modules": true}

var allowedExt = regexp.MustCompile("(?i)(.md|.html|.htm)$")

func shouldSkipDir(name string) bool {
	if name == "." {
		return false
	}

	return strings.HasPrefix(name, ".") || excludedDirs[name]
}

func FileList(fileSystem fs.FS, path string) ([]string, error) {
	var files []string

	err := fs.WalkDir(fileSystem, path, func(path string, d fs.DirEntry, err error) error {
		name := d.Name()

		if err != nil {
			if errors.Is(err, fs.ErrPermission) {
				return nil
			}
			return nil
		}

		// skip hidden and some other dirs
		if d.IsDir() && shouldSkipDir(name) {
			return filepath.SkipDir
		}

		if d.IsDir() || name == "." {
			return nil
		}

		if !allowedExt.MatchString(name) {
			return nil
		}

		files = append(files, path)
		return nil
	})

	return files, err
}
