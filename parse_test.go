package imagesync

import (
	"fmt"
	"imagesync/testutils"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func mdImages(root string) map[string]string {
	var j = filepath.Join
	return map[string]string{
		`![alt text](./assets/subfolder/image.png)`:                    j(root, "./assets/subfolder/image.png"),
		`![](no-alt-text.png)`:                                         j(root, "./no-alt-text.png"),
		`![alt text](assets/img2.jpeg "image title")`:                  j(root, "./assets/img2.jpeg"),
		`![alt text](img3.gif "title")`:                                j(root, "./img3.gif"),
		`![alt text](/home/user/notes/assets 2/name with spaces.jpg)`:  "/home/user/notes/assets 2/name with spaces.jpg", // absolute path
		`![alt text](../assets/img4.svg "title")`:                      j(root, "../assets/img4.svg"),
		`![alt text](../../outside_dir/img5.svg)`:                      j(root, "../../outside_dir/img5.svg"),
		`![alt text](./non_latin/изображение.svg)`:                     j(root, "./non_latin/изображение.svg"),
		`![alt text][imgid1] \n[imgid1]: assets/ref_image.png "title"`: j(root, "assets/ref_image.png"),
		"[![video](./assets/img6.png)](https://youtube.com)":           j(root, "./assets/img6.png"),
		"[![](./assets/img7.png)](https://youtube.com)":                j(root, "./assets/img7.png"),
	}
}

func htmlImages(root string) map[string]string {
	var j = filepath.Join
	return map[string]string{
		`<img src="assets/img7.webp" alt="alt text" style="zoom:50%;" />`: j(root, "./assets/img7.webp"),
		`<img src = "../assets/img8.png" alt="alt text" />`:               j(root, "../assets/img8.png"),
		`<img src=img9.png alt="alt text" />`:                             j(root, "img9.png"),
	}
}

const lorem = "Lorem ipsum dolor sit amet"

func makeMarkdown(path string) (string, []string) {
	var markdown = ` # Test file\n## Paragraph\n
		![link to an image](https://somesite.com/picture.png)`

	images := []string{}

	for k, v := range mdImages(path) {
		markdown = markdown + fmt.Sprintf("\n%s\n%s", lorem, k)
		images = append(images, v)
	}

	for k, v := range htmlImages(path) {
		markdown = markdown + fmt.Sprintf("\n%s\n%s", lorem, k)
		images = append(images, v)
	}

	return markdown, images
}

func makeHTML(path string) (string, []string) {
	var html = ""
	images := []string{}

	for k, v := range htmlImages(path) {
		html += fmt.Sprintf("\n%s\n%s", lorem, k)
		images = append(images, v)
	}

	return html, images
}

func TestGetImagesFromFile(t *testing.T) {

	t.Run("markdown", func(t *testing.T) {
		markdown, want := makeMarkdown("/home/user/notes/my_notes")

		fileSystem := fstest.MapFS{
			"my_notes/note.md": {Data: []byte(markdown)},
		}

		got, err := GetImagesFromFile(fileSystem, "my_notes/note.md", "/home/user/notes")
		if err != nil {
			t.Fatal(err)
		}
		testutils.Compare(t, got, want)
	})

	t.Run("html", func(t *testing.T) {
		html, want := makeHTML("/home/user/pages/my_pages")

		fileSystem := fstest.MapFS{
			"my_pages/page.html": {Data: []byte(html)},
		}

		got, err := GetImagesFromFile(fileSystem, "my_pages/page.html", "/home/user/pages")
		if err != nil {
			t.Fatal(err)
		}
		testutils.Compare(t, got, want)
	})
}
