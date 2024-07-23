package sync

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/flytaly/imagesync/testutils"
	"github.com/stretchr/testify/assert"
)

const lorem = "Lorem ipsum dolor sit amet"

func createMarkdown(basepath string, testlinks []imageTestCase) (string, []LinkInfo) {
	markdown := "# Test file\n## Paragraph\n" + "![link to an image](https://somesite.com/picture.png)"

	images := []LinkInfo{}

	for _, testlink := range testlinks {
		absPath := testlink.link
		absPath = decodePath(absPath)
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Join(basepath, absPath)
		}
		images = append(images, LinkInfo{rootPath: absPath, path: testlink.link, fullLink: testlink.content})
		markdown = markdown + fmt.Sprintf("\n%s\n%s", lorem, testlink.md)
	}

	return markdown, images
}

func TestGetLinksFromFile(t *testing.T) {
	t.Run("markdown images", func(t *testing.T) {
		for i, testcase := range mdImageCases {
			t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
				base := "/home/user/notes/my_notes/"
				markdown, want := createMarkdown(base, []imageTestCase{testcase})
				_, got := GetLinksFromFile(base+"note.md", markdown)
				assert.Equal(t, want, got)
			})
		}
	})

	t.Run("markdown links", func(t *testing.T) {
		for i, testcase := range mdLinkCases {
			t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
				base := "/home/user/notes/my_notes/"
				markdown, want := createMarkdown(base, []imageTestCase{testcase})
				got, _ := GetLinksFromFile(base+"note.md", markdown)
				assert.Equal(t, want, got)
			})
		}
	})
}

func TestReplaceImageLinks(t *testing.T) {
	filePath := "/home/user/notes/my_note/note.md"
	dir := filepath.Dir(filePath)
	j := filepath.Join

	type Move struct {
		from string
		to   string
		link string
	}
	images := []struct {
		move     Move
		linkFrom string
		linkTo   string
	}{
		{
			Move{j(dir, "image1.png"), j(dir, "image2.png"), "image1.png"},
			"![](image1.png)",
			"![](image2.png)",
		},
		{
			Move{j(dir, "./assets/image2.gif"), j(dir, "./assets/subfolder/i.png"), "./assets/images2.gif"},
			`![alt text](./assets/images2.gif "title")`,
			`![alt text](assets/subfolder/i.png "title")`,
		},
		{
			// Absolute path
			Move{j(dir, "image3.png"), j(dir, "image4.png"), j(dir, "image3.png")},
			fmt.Sprintf(`![alt text](%s "title")`, j(dir, "image3.png")),
			fmt.Sprintf(`![alt text](%s "title")`, j(dir, "image4.png")),
		},
		{
			Move{j(dir, "../../pics/pic1.png"), j(dir, "../folder/pic1.png"), "../pics/pic1.png"},
			`<img src = "../pics/pic1.png" alt="alt text" />`,
			`<img src = "../folder/pic1.png" alt="alt text" />`,
		},
	}

	for i, v := range images {
		t.Run(fmt.Sprintf("Replace case %d", i), func(t *testing.T) {
			md := fmt.Sprintf("# Test markdown %d\n## Text with images\n%s\ntext after the image...", i, v.linkFrom)
			want := fmt.Sprintf("# Test markdown %d\n## Text with images\n%s\ntext after the image...", i, v.linkTo)

			link := []MovedLink{{to: v.move.to, link: LinkInfo{rootPath: v.move.from, path: v.move.link, fullLink: v.linkFrom}}}
			got := string(ReplaceImageLinks(filePath, []byte(md), link))
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
