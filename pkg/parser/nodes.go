package parser

type Container interface {
	GetContent() []byte // Markdown content of the block nodes
}

type Paragraph struct {
	Content []byte
}

func (p Paragraph) GetContent() []byte {
	return p.Content
}

type Node interface {
	GetContent() []byte
	GetLiteral() []byte
}

// Leaf is a type of node that cannot have children
type Leaf struct {
	Literal []byte // Text contents of the leaf nodes
	Content []byte // Markdown content of the block nodes
}

func (l Leaf) GetContent() []byte {
	return l.Content
}

func (l Leaf) GetLiteral() []byte {
	return l.Literal
}

type CodeBlock struct {
	Leaf
}

type Code struct {
	Leaf
}

// Text represents markdown text node
type Text struct {
	Leaf
}

// HTMLSpan represents markdown html span node
type HTMLSpan struct {
	Leaf
}

// Link represents markdown link node
type Link struct {
	Leaf

	Destination []byte // Destination is what goes into a href
	Title       []byte // Title is the tooltip thing that goes in a title attribute
}

// Image represents markdown image node
type Image struct {
	Leaf

	Destination []byte // Destination is what goes into a href
	Title       []byte // Title is the tooltip thing that goes in a title attribute
}
