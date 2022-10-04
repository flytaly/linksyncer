package imagesync

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
)

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
	iSync := New(mapFS, ".")
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
		iSync := New(fs, ".")
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
		iSync := New(fs, ".")
		iSync.Files = filesWithLinks
		iSync.Images = linkedFiles

		assert.Contains(t, iSync.Files, note2)
		iSync.RemoveFile(note2)
		assert.NotContains(t, iSync.Files, note2)

		assert.Equal(t, iSync.Images[img1], map[string]struct{}{note1: {}})
		assert.Equal(t, iSync.Images[img2], map[string]struct{}{note1: {}})
	})

}

func TestMoveFile(t *testing.T) {
	fs, filesWithLinks, linkedFiles := GetTestFileSys()
	iSync := New(fs, ".")
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
	iSync := New(fs, ".")
	iSync.Files = filesWithLinks
	iSync.Images = linkedFiles

	note := "notes/folder/note.md"
	// note2 := "notes/folder/note2.md"

	imgs := []MovedLink{{
		from: "notes/folder/assets/image01.png",
		to:   "notes/imgs/renamed.png",
		link: "./assets/image01.png",
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
		assert.NotContains(t, iSync.Images, img.from, "should remove old path")
		assert.Contains(t, iSync.Images, img.to, "should add new path")
		assert.Contains(t, iSync.Images[img.to], note, "should have reference to the source file")
	}

	assert.Contains(t, iSync.Files[note],
		LinkInfo{rootPath: "notes/imgs/renamed.png", originalLink: "../imgs/renamed.png"},
		"should contain updated link")
	assert.NotContains(t, iSync.Files[note],
		LinkInfo{rootPath: imgs[0].from, originalLink: imgs[0].link},
		"old link should be removed")
}

func TestSync(t *testing.T) {
	fs, filesWithLinks, linkedFiles := GetTestFileSys()
	iSync := New(fs, ".")
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
				rootPath:     "notes/index_assets/index.png",
				originalLink: "index_assets/index.png",
			}}, iSync.Files[staticNote])
		}

		if assert.Contains(t, *written, staticNote, "should write updated body") {
			assert.Equalf(t, "![alt text](index_assets/index.png)", (*written)[staticNote], "should update links in the %s's", staticNote)
		}
	})

}
