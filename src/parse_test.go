package imagesync

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
	"testing/fstest"
)

var j = filepath.Join

func mdImages(root string) map[string]string {
	return map[string]string{
		`![alt text](./assets/subfolder/image.png)`:                    j(root, "./assets/subfolder/image.png"),
		`![alt text](assets/img2.jpeg "image title")`:                  j(root, "./assets/img2.jpeg"),
		`![alt text](img3.gif "title")`:                                j(root, "./img3.gif"),
		`![alt text](/home/user/notes/assets 2/name with spaces.jpg)`:  "/home/user/notes/assets 2/name with spaces.jpg", // absolute path
		`![alt text](../assets/img4.svg "title")`:                      j(root, "../assets/img4.svg"),
		`![alt text](../../outside_dir/img5.svg)`:                      j(root, "../../outside_dir/img5.svg"),
		`![alt text](./non_latin/изображение.svg)`:                     j(root, "./non_latin/изображение.svg"),
		`![alt text][imgid1] \n[imgid1]: assets/ref_image.png "title"`: j(root, "assets/ref_image.png"),
		"[![video](./assets/img6.png)](https://youtube.com)":           j(root, "./assets/img6.png"),
	}
}

func htmlImages(root string) map[string]string {
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
		markdown, want := makeMarkdown("notes")

		fileSystem := fstest.MapFS{
			"notes/note.md": {Data: []byte(markdown)},
		}

		got, err := GetImagesFromFile(fileSystem, "notes/note.md")
		if err != nil {
			t.Fatal(err)
		}
		compare(t, got, want)
	})

	t.Run("html", func(t *testing.T) {
		html, want := makeHTML("pages")

		fileSystem := fstest.MapFS{
			"pages/page.html": {Data: []byte(html)},
		}

		got, err := GetImagesFromFile(fileSystem, "pages/page.html")
		if err != nil {
			t.Fatal(err)
		}
		compare(t, got, want)
	})
}

func compare(t *testing.T, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
		t.Errorf("difference %+v", difference(got, want))
	}
}

func difference(slice1, slice2 []string) []string {
	diff := []string{}
	m := map[string]int{}

	for _, v := range slice1 {
		m[v] = 1
	}
	for _, v := range slice2 {
		m[v] = m[v] + 1
	}

	for k, v := range m {
		if v == 1 {
			diff = append(diff, k)
		}
	}

	return diff
}
