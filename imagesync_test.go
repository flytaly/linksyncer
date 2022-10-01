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
	from := "original_name.md"
	to := "new_name.md"
	note2 := "note2.md"
	img1 := LinkInfo{rootPath: "assets/i1.png", originalLink: "./assets/i1.png"}
	img2 := LinkInfo{rootPath: "assets/i2.png", originalLink: "./assets/i2.png"}

	iSync := New(fstest.MapFS{"new_name.md": {Data: []byte("")}}, ".")

	iSync.Files = map[string][]LinkInfo{from: {img1, img2}, note2: {img2}}
	iSync.Images = map[string][]string{
		"assets/i1.png": {from},
		"assets/i2.png": {from, note2},
	}

	extractImagesOriginal := extractImages

	extractImages = func(filePath, content string) []LinkInfo {
		return []LinkInfo{img1, img2}
	}

	iSync.RenameFile(from, to)

	extractImages = extractImagesOriginal

	wantFiles := map[string][]LinkInfo{to: {img1, img2}, note2: {img2}}

	assert.Equal(t, wantFiles, iSync.Files)

	wantImages := map[string][]string{
		"assets/i1.png": {to},
		"assets/i2.png": {note2, to},
	}

	assert.Equal(t, wantImages, iSync.Images)
}

func TestUpdateImageLinks(t *testing.T) {
	filePath := "my_note.md"

	imgs := []RenamedImage{{
		prevPath: "assets/image01.png",
		newPath:  "images/image01.png",
		link:     "./assets/image01.png",
	}}

	content := []byte("![alt text](" + imgs[0].link + ")")

	mapFS := fstest.MapFS{filePath: {Data: content}}

	fileWriterOriginal := writeFile
	t.Cleanup(func() {
		writeFile = fileWriterOriginal
	})

	t.Run("update image links in the file", func(t *testing.T) {
		iSync := New(mapFS, ".")
		iSync.AddFile(filePath)
		writtenData := ""

		writeFile = func(fPath string, data []byte) error {
			writtenData = string(data)
			return nil
		}

		err := iSync.UpdateImageLinks(filePath, imgs)

		if err != nil {
			t.Error(err)
		}

		want := "![alt text](images/image01.png)"
		assert.Equal(t, want, writtenData)
	})

	t.Run("update images in the imagesync struct", func(t *testing.T) {
		iSync := New(mapFS, ".")
		iSync.AddFile(filePath)

		writeFile = func(fPath string, data []byte) error {
			return nil
		}

		err := iSync.UpdateImageLinks(filePath, imgs)
		if err != nil {
			t.Error(err)
		}

		updatedImageList := map[string][]string{}

		for _, img := range imgs {
			updatedImageList[img.newPath] = []string{filePath}
		}

		assert.Equal(t, updatedImageList, iSync.Images)

	})
}
