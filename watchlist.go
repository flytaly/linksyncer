package imagesync

import (
	"io/fs"
	"log"
	"path/filepath"
	"regexp"
	"strings"
)

var allowedExt = regexp.MustCompile("(?i)(" + ValidFilesExtensions + "|" + ImgExtensions + ")$")

func shouldSkipDir(name string) bool {
	if name == "." {
		return false
	}

	return strings.HasPrefix(name, ".") || ExcludedDirs[name]
}

type PathList = map[string]fs.FileInfo

// Returns a map of files and directories which should be watched for changes
func WatchList(fileSystem fs.FS, path string) (PathList, error) {
	var files = PathList{}

	err := fs.WalkDir(fileSystem, path, func(path string, d fs.DirEntry, err error) error {
		name := d.Name()

		if err != nil {
			log.Println(err)
			return nil
		}

		// skip hidden and some other dirs
		if d.IsDir() && shouldSkipDir(name) {
			return filepath.SkipDir
		}

		if name == "." {
			return nil
		}

		if !d.IsDir() && !allowedExt.MatchString(name) {
			return nil
		}

		info, err := d.Info()

		if err != nil {
			return nil
		}

		files[path] = info

		return nil
	})

	return files, err
}
