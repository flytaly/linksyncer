package parser

import (
	"bytes"
	"slices"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func extractImgAndLinks(nodes []*html.Node) []*html.Node {
	result := make([]*html.Node, 0)
	slices.Reverse(nodes)
	for stack := nodes; len(stack) > 0; {
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if node.DataAtom == atom.Img || node.DataAtom == atom.A {
			result = append(result, node)
			continue
		}
		for child := node.LastChild; child != nil; child = child.PrevSibling {
			stack = append(stack, child)
		}
	}
	return result
}

func (p *Parser) appendHTMLFragment(frag []byte) {
	fakeBody := &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}

	entrNodes, err := html.ParseFragment(bytes.NewReader(frag), fakeBody)
	if err != nil {
		return
	}

	for _, node := range extractImgAndLinks(entrNodes) {
		switch node.DataAtom {
		case atom.Img:
			link := &Image{Leaf: Leaf{Content: frag}}
			for _, attr := range node.Attr {
				if attr.Key == "src" {
					link.Destination = []byte(attr.Val)
				}
			}
			if len(link.Destination) == 0 {
				continue
			}
			p.AppendNode(link)
		case atom.A:
			link := &Link{Leaf: Leaf{Content: frag}}
			for _, attr := range node.Attr {
				if attr.Key == "href" {
					link.Destination = []byte(attr.Val)
				}
			}
			if len(link.Destination) == 0 {
				continue
			}
			p.AppendNode(link)
		}

	}
}
