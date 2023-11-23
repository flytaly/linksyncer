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
		var data = []linkFlat{
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
}
