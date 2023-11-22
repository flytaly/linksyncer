package parser

import (
	"testing"
)

func TestParse(t *testing.T) {
	t.Run("parse", func(t *testing.T) {
		p := New()
		p.Parse([]byte("[text](<./some file.png> \"title\")"))

		want := Link{
			Destination: []byte("./some file.png"),
			Title:       []byte("title"),
		}
		var got Link
		for _, v := range p.Nodes {
			if found, ok := v.(*Link); ok {
				got = *found
				break
			}
		}

		if string(got.Destination) != string(want.Destination) {
			t.Errorf("got %q, want %q", got.Destination, want.Destination)
		}
	})
}
