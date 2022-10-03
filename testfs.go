package imagesync

import (
	"testing/fstest"
)

type testFileType int

const (
	parsable testFileType = iota
	image
	skip
)

type testFile = struct {
	mapFile   *fstest.MapFile
	fType     testFileType
	hasLinks  []LinkInfo
	wasLinked []string
}

var testFiles = map[string]testFile{
	"notes/.git/ignored-file.md": {
		mapFile: &fstest.MapFile{},
		fType:   skip,
	},
	"notes/folder/note.md": {
		mapFile: &fstest.MapFile{Data: []byte(
			`![alt text](./assets/image01.png)
             ![alt text](./assets/image02.png)`)},
		fType: parsable,
		hasLinks: []LinkInfo{
			{rootPath: "notes/folder/assets/image01.png", originalLink: "./assets/image01.png"},
			{rootPath: "notes/folder/assets/image02.png", originalLink: "./assets/image02.png"},
		},
	},
	"notes/folder/note2.md": {
		mapFile:  &fstest.MapFile{Data: []byte("![alt text](./assets/image02.png)")},
		fType:    parsable,
		hasLinks: []LinkInfo{{rootPath: "notes/folder/assets/image02.png", originalLink: "./assets/image02.png"}},
	},
	"notes/folder/assets/image01.png": {
		mapFile:   &fstest.MapFile{},
		fType:     image,
		wasLinked: []string{"notes/folder/note.md"},
	},
	"notes/folder/assets/image02.png": {
		mapFile:   &fstest.MapFile{},
		fType:     image,
		wasLinked: []string{"notes/folder/note.md", "notes/folder/note2.md"},
	},
}

func GetTestFileSys() (fs fstest.MapFS, links map[string][]LinkInfo, wasLinked map[string]map[string]struct{}) {
	fs = make(map[string]*fstest.MapFile)
	links = make(map[string][]LinkInfo)
	wasLinked = make(map[string]map[string]struct{})

	for path, testFile := range testFiles {
		fs[path] = testFile.mapFile
		if len(testFile.hasLinks) > 0 {
			links[path] = testFile.hasLinks
		}
		if len(testFile.wasLinked) > 0 {
			wasLinked[path] = map[string]struct{}{}
			for _, v := range testFile.wasLinked {
				wasLinked[path][v] = struct{}{}
			}
		}
	}

	return fs, links, wasLinked
}
