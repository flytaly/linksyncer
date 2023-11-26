package parser

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

type linkFlat struct {
	dest  string
	title string
	md    string
}

func toLinkFlat(l Link) linkFlat {
	return linkFlat{string(l.Destination), string(l.Title), string(l.Leaf.Content)}
}

func TestParse(t *testing.T) {
	t.Run("Normal Links", func(t *testing.T) {
		data := []linkFlat{
			{"./some file.md", "title", "[text](<./some file.md> \"title\")"},
			{"./with (parenthesis).md", "title", "[text](./with (parenthesis).md \"title\")"},
			{"./foo.md", "(bar)", "[[1]](./foo.md '(bar)')"},
		}

		for i, want := range data {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				p := New()
				p.Parse([]byte(want.md))
				got, _ := p.LinksAndImages()
				if len(got) != 1 {
					t.Fatalf("should be exactly one link, got %d", len(got))
				}
				assert.Equal(t, want, toLinkFlat(got[0]))
			})
		}
	})

	t.Run("Normal images", func(t *testing.T) {
		tests := []struct {
			md   string
			want linkFlat
		}{
			{
				"![alt text](<./some file.png> \"title\")",
				linkFlat{"./some file.png", "title", "[alt text](<./some file.png> \"title\")"},
			},
		}

		for i, tt := range tests {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				p := New()
				p.Parse([]byte(tt.md))
				_, got := p.LinksAndImages()
				if len(got) != 1 {
					t.Fatalf("should be exactly one image link, got %d", len(got))
				}
				assert.Equal(t, tt.want, toLinkFlat(Link(got[0])))
			})
		}
	})

	t.Run("Ref links", func(t *testing.T) {
		md := "[text][ref]\n\n[ref]: ref_file.md \"title\""
		l := linkFlat{"ref_file.md", "title", "[ref]: ref_file.md"}

		p := New()
		p.Parse([]byte(md))
		got, _ := p.LinksAndImages()
		if len(got) != 1 {
			t.Fatalf("should be exactly one link, got %d", len(got))
		}
		assert.Equal(t, l, toLinkFlat(got[0]))
	})

	t.Run("Ref images", func(t *testing.T) {
		md := "![alt text][ref_img]\n\n[ref_img]: ref_image.png \"my image\""
		l := linkFlat{"ref_image.png", "my image", "[ref_img]: ref_image.png"}

		p := New()
		p.Parse([]byte(md))
		_, got := p.LinksAndImages()
		if len(got) != 1 {
			t.Fatalf("should be exactly one link, got %d", len(got))
		}
		assert.Equal(t, l, toLinkFlat(Link(got[0])))
	})

	t.Run("html_link", func(t *testing.T) {
		a := `<a href="./note2.md">`
		md := "<p class=\"c\">" + a + "link</a></p>"
		l := linkFlat{"./note2.md", "", a}

		p := New()
		p.Parse([]byte(md))
		got, _ := p.LinksAndImages()
		if len(got) != 1 {
			t.Fatalf("should be exactly one link, got %d", len(got))
		}
		assert.Equal(t, l, toLinkFlat(Link(got[0])))
	})

	t.Run("html_images", func(t *testing.T) {
		img := `<img width="400px" src="./assets/image.png" title="title" />`
		md := "<p class=\"c\">" + img + "</p>"
		l := linkFlat{"./assets/image.png", "", img}

		p := New()
		p.Parse([]byte(md))
		_, got := p.LinksAndImages()
		if len(got) != 1 {
			t.Fatalf("should be exactly one link, got %d", len(got))
		}
		assert.Equal(t, l, toLinkFlat(Link(got[0])))
	})
}
