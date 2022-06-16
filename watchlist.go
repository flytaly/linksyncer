package imagesync

import (
	"io/fs"
	"log"
	"path/filepath"
	"regexp"
	"strings"
)

var allowedExt = regexp.MustCompile("(?i)(" + ValidFilesExtensions + ")$")

func shouldSkipDir(name string) bool {
	if name == "." {
		return false
	}

	return strings.HasPrefix(name, ".") || ExcludedDirs[name]
}

type PathList = map[string]fs.FileInfo

// Returns a map of files and directories which should be watched for changes
func WatchList(fileSystem fs.FS, path string) (dirs []string, files []string, err error) {
	err = fs.WalkDir(fileSystem, path, func(path string, d fs.DirEntry, err error) error {
		name := d.Name()

		if err != nil {
			log.Println(err)
			return nil
		}

		if name == "." {
			return nil
		}

		if d.IsDir() {
			// skip hidden and some other dirs
			if shouldSkipDir(name) {
				return filepath.SkipDir
			}
			dirs = append(dirs, path)
			return nil
		}

		if allowedExt.MatchString(name) {
			files = append(files, path)
		}

		return nil
	})

	return dirs, files, err
}
