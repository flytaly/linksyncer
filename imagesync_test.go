package imagesync

import (
	"imagesync/pkg/fswatcher"
	"imagesync/pkg/log"
	"io/fs"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
)

func NewTestISync(fs fs.FS, root string) *ImageSync {
	return New(fs, root, func(is *ImageSync) {
		is.log = log.NewEmptyLog()
	})
}

func mockWriteFile(t *testing.T) (*map[string]string, func()) {
	t.Helper()
	original := writeFile

	writes := map[string]string{}
	writeFile = func(fPath string, data []byte) error {
		writes[fPath] = string(data)
		return nil
	}
	restore := func() {
		writeFile = original
	}
	return &writes, restore
}

func TestProcessFiles(t *testing.T) {
	mapFS, wantFiles, wantRefs := GetTestFileSys()
	iSync := NewTestISync(mapFS, "notes")
	iSync.ProcessFiles()
	assert.Equal(t, wantFiles, iSync.Files)
	assert.Equal(t, wantRefs, iSync.Images)
}

func TestRemoveFile(t *testing.T) {
	note1 := "notes/folder/note.md"
	note2 := "notes/folder/note2.md"
	img1 := "notes/folder/assets/image01.png"
	img2 := "notes/folder/assets/image02.png"
	t.Run("remove note 1", func(t *testing.T) {
		fs, filesWithLinks, linkedFiles := GetTestFileSys()
		iSync := NewTestISync(fs, ".")
		iSync.Files = filesWithLinks
		iSync.Images = linkedFiles

		assert.Contains(t, iSync.Files, note1)
		iSync.RemoveFile(note1)
		assert.NotContains(t, iSync.Files, note1)

		assert.NotContains(t, iSync.Images, img1)
		assert.Equal(t, iSync.Images[img2], map[string]struct{}{note2: {}})
	})

	t.Run("remove note 2", func(t *testing.T) {
		fs, filesWithLinks, linkedFiles := GetTestFileSys()
		iSync := NewTestISync(fs, ".")
		iSync.Files = filesWithLinks
		iSync.Images = linkedFiles

		assert.Contains(t, iSync.Files, note2)
		iSync.RemoveFile(note2)
		assert.NotContains(t, iSync.Files, note2)

		assert.Equal(t, iSync.Images[img1], map[string]struct{}{note1: {}})
		assert.Equal(t, iSync.Images[img2], map[string]struct{}{note1: {}})
	})

}

func TestUpdateFile(t *testing.T) {
	t.Run("clear content", func(t *testing.T) {
		fs, filesWithLinks, linkedFiles := GetTestFileSys()
		iSync := NewTestISync(fs, ".")
		iSync.Files = filesWithLinks
		iSync.Images = linkedFiles

		file := "notes/folder/note.md"
		fs[file] = &fstest.MapFile{Data: []byte("")}
		iSync.UpdateFile(file)
		assert.Equal(t, iSync.Files[file], []LinkInfo{})
	})

	t.Run("update file", func(t *testing.T) {
		fs, filesWithLinks, linkedFiles := GetTestFileSys()
		iSync := NewTestISync(fs, ".")
		iSync.Files = filesWithLinks
		iSync.Images = linkedFiles

		file := "notes/folder/note.md"
		fs[file] = &fstest.MapFile{Data: []byte(`![alt text](./assets/image.png)`)}
		iSync.UpdateFile(file)

		newLinks := []LinkInfo{
			{rootPath: "notes/folder/assets/image.png", path: "./assets/image.png", fullLink: "![alt text](./assets/image.png)"}}
		assert.Equal(t, iSync.Files[file], newLinks)
	})

}

func TestMoveFile(t *testing.T) {
	fs, filesWithLinks, linkedFiles := GetTestFileSys()
	iSync := NewTestISync(fs, ".")
	iSync.Files = filesWithLinks
	iSync.Images = linkedFiles

	from := "notes/folder/note.md"
	to := "notes/renamed.md"

	linkedFile1 := "notes/folder/assets/image01.png"
	linkedFile2 := "notes/folder/assets/image02.png"

	fs[to] = &fstest.MapFile{Data: fs[from].Data}
	delete(fs, from)

	gotData, restore := mockWriteFile(t)
	t.Cleanup(func() { restore() })

	iSync.MoveFile(from, to, nil)

	assert.NotContains(t, iSync.Files, from, "should delete old path")
	assert.Contains(t, iSync.Files, to, "should add new path")

	assert.NotContains(t, iSync.Images[linkedFile1], from, "should delete old reference")
	assert.Contains(t, iSync.Images[linkedFile1], to, "should add new reference")
	assert.NotContains(t, iSync.Images[linkedFile2], from, "should delete old reference")
	assert.Contains(t, iSync.Images[linkedFile2], to, "should add new reference")

	expected := map[string]string{
		to: `![alt text](folder/assets/image01.png)\n![alt text](folder/assets/image02.png)`,
	}
	assert.Equal(t, expected, *gotData)
}

func TestUpdateImageLinks(t *testing.T) {
	fs, filesWithLinks, linkedFiles := GetTestFileSys()
	iSync := NewTestISync(fs, ".")
	iSync.Files = filesWithLinks
	iSync.Images = linkedFiles

	note := "notes/folder/note.md"
	// note2 := "notes/folder/note2.md"

	imgs := []MovedLink{{
		to:   "notes/imgs/renamed.png",
		link: LinkInfo{rootPath: "notes/folder/assets/image01.png", path: "./assets/image01.png", fullLink: "![alt text](./assets/image01.png)"},
	}}

	written, restore := mockWriteFile(t)
	t.Cleanup(func() { restore() })

	err := iSync.UpdateLinksInFile(note, imgs)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]string{
		note: `![alt text](../imgs/renamed.png)\n![alt text](./assets/image02.png)`,
	}
	assert.Equal(t, want, *written, "image links in the file should be updated")

	for _, img := range imgs {
		assert.NotContains(t, iSync.Images, img.link.rootPath, "should remove old path")
		assert.Contains(t, iSync.Images, img.to, "should add new path")
		assert.Contains(t, iSync.Images[img.to], note, "should have reference to the source file")
	}

	assert.Contains(t, iSync.Files[note],
		LinkInfo{rootPath: "notes/imgs/renamed.png", path: "../imgs/renamed.png", fullLink: "![alt text](../imgs/renamed.png)"},
		"should contain updated link")
	assert.NotContains(t, iSync.Files[note],
		imgs[0],
		"old link should be removed")
}

func TestSync(t *testing.T) {
	t.Run("t1", func(t *testing.T) {
		fs, filesWithLinks, linkedFiles := GetTestFileSys()
		iSync := NewTestISync(fs, ".")
		iSync.Files = filesWithLinks
		iSync.Images = linkedFiles

		movedNotes := []struct {
			from string
			to   string
			body string
		}{
			{
				from: "notes/folder/note.md",
				to:   "notes/n/renamed.md",
				body: `![alt text](../images/image01.png)\n![alt text](../folder/assets/image02.png)`,
			},
			{
				from: "notes/folder/note2.md",
				to:   "notes/folder/note3.md",
				body: "![alt text](assets/image02.png)",
			},
		}
		links := map[string]string{
			"notes/folder/assets/image01.png": "notes/images/image01.png",
			"notes/index.png":                 "notes/index_assets/index.png",
		}
		moves := make(map[string]string)
		for _, v := range movedNotes {
			moves[v.from] = v.to
			fs[v.to] = &fstest.MapFile{Data: fs[v.from].Data}
		}
		for k, v := range links {
			moves[k] = v
		}

		written, restore := mockWriteFile(t)
		t.Cleanup(func() { restore() })

		iSync.Sync(moves)

		t.Run("check info of the moved notes in the cache", func(t *testing.T) {
			for _, n := range movedNotes {
				assert.NotContains(t, iSync.Files, n.from, "should remove old path")
				assert.Contains(t, iSync.Files, n.to, "should add new path")

				assert.Equalf(t, n.body, (*written)[n.to], "should update links in the %s's", n.to)
			}
		})

		t.Run("test linked files in cache", func(t *testing.T) {
			for from, to := range links {
				assert.NotContains(t, iSync.Images, from, "old path to linked files should be removed")
				assert.Contains(t, iSync.Images, to, "new path to linked files should be saved")
			}

			assert.Equal(t, map[string]struct{}{
				movedNotes[0].to: {},
			}, iSync.Images["notes/images/image01.png"])

		})

		t.Run("check info of the static note in the cache", func(t *testing.T) {
			staticNote := "notes/index.md"

			if assert.Contains(t, iSync.Files, staticNote) {
				assert.Equal(t, []LinkInfo{{
					rootPath: "notes/index_assets/index.png",
					path:     "index_assets/index.png",
					fullLink: "![alt text](index_assets/index.png)",
				}}, iSync.Files[staticNote])
			}

			if assert.Contains(t, *written, staticNote, "should write updated body") {
				assert.Equalf(t, "![alt text](index_assets/index.png)", (*written)[staticNote], "should update links in the %s's", staticNote)
			}
		})

	})

	t.Run("duplicates", func(t *testing.T) {
		var fs fstest.MapFS = make(map[string]*fstest.MapFile)
		from := "notes/rnd/note1.md"
		to := "notes/note1.md"
		fs[from] = &fstest.MapFile{Data: []byte("![](img1.png)\n!Some Text\n![](img1.png)")}
		fs["notes/rnd/img1.jpg"] = &fstest.MapFile{Data: []byte("")}
		iSync := NewTestISync(fs, ".")
		iSync.ProcessFiles()

		gotData, restore := mockWriteFile(t)
		t.Cleanup(func() { restore() })

		fs[to] = fs[from]
		delete(fs, from)
		iSync.Sync(map[string]string{from: to})

		expect := "![](rnd/img1.png)\n!Some Text\n![](rnd/img1.png)"
		assert.Equal(t, expect, (*gotData)[to])
	})
}

func TestWatch(t *testing.T) {
	t.Run("CREATE", func(t *testing.T) {
		var fs fstest.MapFS = make(map[string]*fstest.MapFile)
		iSync := NewTestISync(fs, ".")
		iSync.Watch(time.Second)
		name := "notes/note1.md"
		fs[name] = &fstest.MapFile{Data: []byte("")}
		iSync.Watcher.SendEvent(fswatcher.Event{Name: name, Op: fswatcher.Create})
		assert.Contains(t, iSync.Files, name)
		iSync.Close()
	})

	t.Run("REMOVE", func(t *testing.T) {
		fs, filesWithLinks, linkedFiles := GetTestFileSys()
		iSync := NewTestISync(fs, ".")
		iSync.Files = filesWithLinks
		iSync.Images = linkedFiles
		iSync.Watch(time.Second)
		name := "notes/folder/note.md"
		assert.Contains(t, iSync.Files, name)
		iSync.Watcher.SendEvent(fswatcher.Event{Name: name, Op: fswatcher.Remove})
		iSync.Watch(time.Millisecond * 5)
		assert.NotContains(t, iSync.Files, name)
		iSync.Close()
	})

	t.Run("WRITE", func(t *testing.T) {
		fs, filesWithLinks, linkedFiles := GetTestFileSys()
		iSync := NewTestISync(fs, ".")
		iSync.Files = filesWithLinks
		iSync.Images = linkedFiles
		iSync.Watch(time.Second)
		name := "notes/folder/note.md"
		fs[name] = &fstest.MapFile{Data: []byte("")}
		iSync.Watcher.SendEvent(fswatcher.Event{Name: name, Op: fswatcher.Write})
		iSync.Watch(time.Millisecond * 5)
		assert.Equal(t, []LinkInfo{}, iSync.Files[name])
		iSync.Close()
	})

	t.Run("RENAME", func(t *testing.T) {
		var fs fstest.MapFS = make(map[string]*fstest.MapFile)
		noteFrom := "notes/other/note1.md"
		noteTo := "notes/note1.md"
		imgFrom := "notes/other/image1.png"
		imgTo := "notes/assets/image1.png"
		fs[noteFrom] = &fstest.MapFile{Data: []byte("![](./image1.png)")}
		fs[imgFrom] = &fstest.MapFile{}

		iSync := NewTestISync(fs, ".")
		iSync.ProcessFiles()
		assert.Equal(t, iSync.Files[noteFrom], []LinkInfo{{rootPath: imgFrom, path: "./image1.png", fullLink: "![](./image1.png)"}})

		gotData, restore := mockWriteFile(t)
		t.Cleanup(func() { restore() })

		fs[noteTo] = fs[noteFrom]
		delete(fs, noteFrom)
		fs[imgTo] = fs[imgFrom]
		delete(fs, imgFrom)

		iSync.Watch(time.Millisecond)
		time.Sleep(time.Millisecond * 40)
		iSync.Close()

		expected := map[string]string{noteTo: "![](assets/image1.png)"}
		assert.Equal(t, expected, *gotData)
	})
}
