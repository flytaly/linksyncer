package imagesync

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
)

func mdImages() map[string]string {
	return map[string]string{
		`![alt text](./assets/subfolder/image.png)`:                    "./assets/subfolder/image.png",
		`![](no-alt-text.png)`:                                         "no-alt-text.png",
		`![alt text](assets/img2.jpeg "image title")`:                  "assets/img2.jpeg",
		`![alt text](img3.gif "title")`:                                "img3.gif",
		`![alt text](/home/user/notes/assets 2/name with spaces.jpg)`:  "/home/user/notes/assets 2/name with spaces.jpg", // absolute path
		`![alt text](../assets/img4.svg "title")`:                      "../assets/img4.svg",
		`![alt text](../../outside_dir/img5.svg)`:                      "../../outside_dir/img5.svg",
		`![alt text](./non_latin/изображение.svg)`:                     "./non_latin/изображение.svg",
		`![alt text][imgid1] \n[imgid1]: assets/ref_image.png "title"`: "assets/ref_image.png",
		"[![video](./assets/img6.png)](https://youtube.com)":           "./assets/img6.png",
		"[![](./assets/img7.png)](https://youtube.com)":                "./assets/img7.png",
	}
}

func htmlImages() map[string]string {
	return map[string]string{
		`<img src="assets/img7.webp" alt="alt text" style="zoom:50%;" />`: "assets/img7.webp",
		`<img src = "../assets/img8.png" alt="alt text" />`:               "../assets/img8.png",
		`<img src=img9.png alt="alt text" />`:                             "img9.png",
	}
}

const lorem = "Lorem ipsum dolor sit amet"

func makeMarkdown(basepath string) (string, []ImageInfo) {
	var markdown = ` # Test file\n## Paragraph\n
		![link to an image](https://somesite.com/picture.png)`

	images := []ImageInfo{}

	for k, path := range mdImages() {
		absPath := path
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Join(basepath, path)
		}
		images = append(images, ImageInfo{absPath: absPath, originalLink: path})
		markdown = markdown + fmt.Sprintf("\n%s\n%s", lorem, k)
	}

	for k, path := range htmlImages() {
		absPath := path
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Join(basepath, path)
		}
		images = append(images, ImageInfo{absPath: absPath, originalLink: path})
		markdown = markdown + fmt.Sprintf("\n%s\n%s", lorem, k)
	}

	return markdown, images
}

func makeHTML(basepath string) (string, []ImageInfo) {
	var html = ""
	images := []ImageInfo{}

	for k, path := range htmlImages() {
		absPath := path
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Join(basepath, path)
		}
		images = append(images, ImageInfo{absPath: absPath, originalLink: path})
		html += fmt.Sprintf("\n%s\n%s", lorem, k)
	}

	return html, images
}

func TestGetImagesFromFile(t *testing.T) {

	t.Run("markdown", func(t *testing.T) {
		markdown, want := makeMarkdown("/home/user/notes/my_notes")

		got := GetImagesFromFile("/home/user/notes/my_notes/note.md", markdown)

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got: %v,\n want: %v\n", got, want)
		}
	})

	t.Run("html", func(t *testing.T) {
		html, want := makeHTML("/home/user/pages/my_pages")

		got := GetImagesFromFile("/home/user/pages/my_pages/page.html", html)

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got: %v,\n want: %v\n", got, want)
		}
	})
}
