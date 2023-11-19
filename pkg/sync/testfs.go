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
		mapFile: &fstest.MapFile{Data: []byte(`![alt text](./assets/image01.png)\n![alt text](./assets/image02.png)`)},
		fType:   parsable,
		hasLinks: []LinkInfo{
			{rootPath: "notes/folder/assets/image01.png", path: "./assets/image01.png", fullLink: "![alt text](./assets/image01.png)"},
			{rootPath: "notes/folder/assets/image02.png", path: "./assets/image02.png", fullLink: "![alt text](./assets/image02.png)"},
		},
	},
	"notes/folder/note2.md": {
		mapFile:  &fstest.MapFile{Data: []byte("![alt text](./assets/image02.png)")},
		fType:    parsable,
		hasLinks: []LinkInfo{{rootPath: "notes/folder/assets/image02.png", path: "./assets/image02.png", fullLink: "![alt text](./assets/image02.png)"}},
	},
	"notes/index.md": {
		mapFile:  &fstest.MapFile{Data: []byte("![alt text](./index.png)")},
		fType:    parsable,
		hasLinks: []LinkInfo{{rootPath: "notes/index.png", path: "./index.png", fullLink: "![alt text](./index.png)"}},
	},
	"notes/инфо.md": {
		mapFile:  &fstest.MapFile{Data: []byte("![alt text](./%D0%BA%D0%B0%D1%80%D1%82%D0%B8%D0%BD%D0%BA%D0%B0.png)")},
		fType:    parsable,
		hasLinks: []LinkInfo{{rootPath: "notes/картинка.png", path: "./%D0%BA%D0%B0%D1%80%D1%82%D0%B8%D0%BD%D0%BA%D0%B0.png", fullLink: "![alt text](./%D0%BA%D0%B0%D1%80%D1%82%D0%B8%D0%BD%D0%BA%D0%B0.png)"}},
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
	"notes/index.png": {
		mapFile:   &fstest.MapFile{},
		fType:     image,
		wasLinked: []string{"notes/index.md"},
	},
	"notes/картинка.png": {
		mapFile:   &fstest.MapFile{},
		fType:     image,
		wasLinked: []string{"notes/инфо.md"},
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
