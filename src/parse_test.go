package imagesync

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
	"testing/fstest"
)

func getTestImages(root string) map[string]string {
	j := filepath.Join
	return map[string]string{
		`![alt text](./assets/subfolder/image.png)`:                    j(root, "./assets/subfolder/image.png"),
		`![alt text](assets/img2.jpeg "image title")`:                  j(root, "./assets/img2.jpeg"),
		`![alt text](img3.gif "title")`:                                j(root, "./img3.gif"),
		`![alt text](/home/user/notes/assets 2/name with spaces.jpg)`:  "/home/user/notes/assets 2/name with spaces.jpg", // absolute path
		`![alt text](../assets/img4.svg "title")`:                      j(root, "../assets/img4.svg"),
		`![alt text](../../outside_dir/img5.svg)`:                      j(root, "../../outside_dir/img5.svg"),
		`![alt text](./non_latin/изображение.svg)`:                     j(root, "./non_latin/изображение.svg"),
		`![alt text][imgid1] \n[imgid1]: assets/ref_image.png "title"`: j(root, "assets/ref_image.png"),
		"[![video](./assets/name.png)](https://youtube.com)":           j(root, "./assets/name.png"),
		// `<img src="assets/img1.webp" alt="alt text" style="zoom:50%;" />`: j(root, "./assets/img1.webp"),
		// `<img src = "../assets/img2.png" alt="alt text" />`:               j(root, "../assets/img2.png"),
		// `< img src = "img3.png" alt="alt text" />`:                        j(root, "img3.png"),
	}
}

var markdown = `
# Test file

## Paragraph

![link to an image](https://somesite.com/picture.png)
`

const lorem = "Lorem ipsum dolor sit amet"

func TestParseMd(t *testing.T) {
	want := []string{}

	for k, v := range getTestImages("notes") {
		markdown = markdown + fmt.Sprintf("\n%s\n%s", lorem, k)
		want = append(want, v)
	}

	fileSystem := fstest.MapFS{
		"notes/note.md": {Data: []byte(markdown)},
	}

	got, err := GetImagesFromFile(fileSystem, "notes/note.md")

	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
		t.Errorf("difference %+v", difference(got, want))
	}

}

func difference(slice1 []string, slice2 []string) []string {
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
