package imagesync

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
)

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
	t.Run("remove note 1", func(t *testing.T) {
		fs, filesWithLinks, linkedFiles := GetTestFileSys()
		iSync := New(fs, ".")
		iSync.Files = filesWithLinks
		iSync.Images = linkedFiles

		assert.Contains(t, iSync.Files, note1)
		iSync.RemoveFile(note1)
		assert.NotContains(t, iSync.Files, note1)

		refsAfter := map[string][]string{
			// "notes/folder/assets/image01.png": {note1},
			"notes/folder/assets/image02.png": { /* note1, */ note2},
		}
		assert.Equal(t, refsAfter, iSync.Images)
	})

	t.Run("remove note 2", func(t *testing.T) {
		fs, filesWithLinks, linkedFiles := GetTestFileSys()
		iSync := New(fs, ".")
		iSync.Files = filesWithLinks
		iSync.Images = linkedFiles

		assert.Contains(t, iSync.Files, note2)
		iSync.RemoveFile(note2)
		assert.NotContains(t, iSync.Files, note2)

		refsAfter := map[string][]string{
			"notes/folder/assets/image01.png": {note1},
			"notes/folder/assets/image02.png": {note1 /* , note2 */},
		}
		assert.Equal(t, refsAfter, iSync.Images)
	})

}

func TestRenameFile(t *testing.T) {
	fs, filesWithLinks, linkedFiles := GetTestFileSys()
	iSync := New(fs, ".")
	iSync.Files = filesWithLinks
	iSync.Images = linkedFiles

	from := "notes/folder/note.md"
	to := "notes/folder/renamed.md"

	linkedFile1 := "notes/folder/assets/image01.png"
	linkedFile2 := "notes/folder/assets/image02.png"

	fs[to] = &fstest.MapFile{Data: fs[from].Data}
	delete(fs, from)

	iSync.RenameFile(from, to)

	assert.NotContains(t, iSync.Files, from)
	assert.Contains(t, iSync.Files, to)

	assert.NotContains(t, iSync.Images[linkedFile1], from)
	assert.Contains(t, iSync.Images[linkedFile1], to)
	assert.NotContains(t, iSync.Images[linkedFile2], from)
	assert.Contains(t, iSync.Images[linkedFile2], to)
}

func TestUpdateImageLinks(t *testing.T) {
	fs, filesWithLinks, linkedFiles := GetTestFileSys()
	iSync := New(fs, ".")
	iSync.Files = filesWithLinks
	iSync.Images = linkedFiles

	note := "notes/folder/note.md"
	// note2 := "notes/folder/note2.md"

	imgs := []RenamedImage{{
		prevPath: "notes/folder/assets/image01.png",
		newPath:  "notes/folder/imgs/renamed.png",
		link:     "./assets/image01.png",
	}}

	writtenData := ""
	writeFile = func(fPath string, data []byte) error {
		assert.Equal(t, note, fPath)
		writtenData = string(data)
		return nil
	}

	err := iSync.UpdateLinksInFile(note, imgs)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("update image links in the files", func(t *testing.T) {
		want := `![alt text](imgs/renamed.png)
             ![alt text](./assets/image02.png)`
		assert.Equal(t, want, writtenData)
	})

	t.Run("update images in the imagesync struct", func(t *testing.T) {
		for _, img := range imgs {
			assert.NotContains(t, iSync.Images, img.prevPath)
			assert.Contains(t, iSync.Images, img.newPath)
			assert.Contains(t, iSync.Images[img.newPath], note)
		}
	})
}
