package imagesync

import (
	"fmt"
	"imagesync/testutils"
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

func makeMarkdown(basepath string) (string, []LinkInfo) {
	var markdown = ` # Test file\n## Paragraph\n
		![link to an image](https://somesite.com/picture.png)`

	images := []LinkInfo{}

	for k, path := range mdImages() {
		absPath := path
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Join(basepath, path)
		}
		images = append(images, LinkInfo{rootPath: absPath, originalLink: path})
		markdown = markdown + fmt.Sprintf("\n%s\n%s", lorem, k)
	}

	for k, path := range htmlImages() {
		absPath := path
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Join(basepath, path)
		}
		images = append(images, LinkInfo{rootPath: absPath, originalLink: path})
		markdown = markdown + fmt.Sprintf("\n%s\n%s", lorem, k)
	}

	return markdown, images
}

func makeHTML(basepath string) (string, []LinkInfo) {
	var html = ""
	images := []LinkInfo{}

	for k, path := range htmlImages() {
		absPath := path
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Join(basepath, path)
		}
		images = append(images, LinkInfo{rootPath: absPath, originalLink: path})
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

func TestReplaceImageLinks(t *testing.T) {
	filePath := "/home/user/notes/my_note/note.md"
	dir := filepath.Dir(filePath)
	j := filepath.Join

	images := []struct {
		MovedLink
		linkFrom string
		linkTo   string
	}{
		{
			MovedLink{j(dir, "image1.png"), j(dir, "image2.png"), "image1.png"},
			"![](image1.png)",
			"![](image2.png)",
		},
		{
			MovedLink{j(dir, "./assets/image2.gif"), j(dir, "./assets/subfolder/i.png"), "./assets/images2.gif"},
			`![alt text](./assets/images2.gif "title")`,
			`![alt text](assets/subfolder/i.png "title")`,
		},
		{
			// Absolute path
			MovedLink{j(dir, "image3.png"), j(dir, "image4.png"), j(dir, "image3.png")},
			fmt.Sprintf(`![alt text](%s "title")`, j(dir, "image3.png")),
			fmt.Sprintf(`![alt text](%s "title")`, j(dir, "image4.png")),
		},
		{
			MovedLink{j(dir, "../../pics/pic1.png"), j(dir, "../folder/pic1.png"), "../pics/pic1.png"},
			`<img src = "../pics/pic1.png" alt="alt text" />`,
			`<img src = "../folder/pic1.png" alt="alt text" />`,
		},
	}

	for i, v := range images {
		t.Run(fmt.Sprintf("Replace case %d", i), func(t *testing.T) {
			md := fmt.Sprintf("# Test markdown %d\n## Text with images\n%s\ntext after the image...", i, v.linkFrom)
			want := fmt.Sprintf("# Test markdown %d\n## Text with images\n%s\ntext after the image...", i, v.linkTo)

			got := string(ReplaceImageLinks(filePath, []byte(md), []MovedLink{v.MovedLink}))
			assertText(t, got, want)
		})
	}
}

func assertText(t testing.TB, got, want string) {
	t.Helper()
	if got != want {
		d1, d2 := testutils.StringDifference(got, want)
		t.Errorf("\n==Got=>\n%s\n==Want=>\n%s\n==Diff=>\n%s\n%s", got, want, d1, d2)
	}
}
