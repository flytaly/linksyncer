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

	t.Run("update image links in the files", func(t *testing.T) {
		want := map[string]string{
			note: `![alt text](../imgs/renamed.png)\n![alt text](./assets/image02.png)`,
		}
		assert.Equal(t, want, *written)
	})

	t.Run("update images in the imagesync struct", func(t *testing.T) {
		for _, img := range imgs {
			assert.NotContains(t, iSync.Images, img.from)
			assert.Contains(t, iSync.Images, img.to)
			assert.Contains(t, iSync.Images[img.to], note)
		}
	})
}

func TestSync(t *testing.T) {
	fs, filesWithLinks, linkedFiles := GetTestFileSys()
	iSync := New(fs, ".")
	iSync.Files = filesWithLinks
	iSync.Images = linkedFiles

	notes := []struct {
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
	for _, v := range notes {
		moves[v.from] = v.to
		fs[v.to] = &fstest.MapFile{Data: fs[v.from].Data}
	}
	for k, v := range links {
		moves[k] = v
	}

	written, restore := mockWriteFile(t)
	t.Cleanup(func() { restore() })

	t.Run("sync moved files", func(t *testing.T) {
		iSync.Sync(moves)
		for _, n := range notes {
			assert.NotContains(t, iSync.Files, n.from, "should remove old path")
			assert.Contains(t, iSync.Files, n.to, "should add new path")
			assert.Equalf(t, n.body, (*written)[n.to], "should update links in the %s's", n.to)
		}

	})

	t.Run("sync not displaced files with moved links", func(t *testing.T) {
		// note moved file with updated links
		note := "notes/index.md"
		assert.Contains(t, iSync.Files, note)
		ok := assert.Contains(t, *written, note, "should write updated body")
		if ok {
			assert.Equalf(t, "![alt text](index_assets/index.png)", (*written)[note], "should update links in the %s's", note)
		}

		t.Error("TODO:")
	})
}
