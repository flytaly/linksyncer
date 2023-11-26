package parser

import (
	"bytes"
)

// Parsing of inline elements

// Inline parses text within a block.
// Each function returns the number of consumed chars.
func (p *Parser) Inline(data []byte) {
	// handlers might call us recursively: enforce a maximum depth
	if p.nesting >= p.maxNesting || len(data) == 0 {
		return
	}
	p.nesting++
	beg, end := 0, 0

	n := len(data)
	for end < n {
		handler := p.inlineCallback[data[end]]
		if handler == nil {
			end++
			continue
		}
		consumed, node := handler(p, data, end)
		if consumed == 0 {
			// no action from the callback
			end++
			continue
		}
		if node != nil {
			if html, ok := node.(*HTMLSpan); ok {
				p.appendHTMLFragment(html.Literal)
			} else {
				p.AppendNode(node)
			}
		}
		// skip inactive chars
		// data[beg:end]
		beg = end + consumed
		end = beg
	}

	p.nesting--
}

func codeSpan(p *Parser, data []byte, offset int) (int, Node) {
	data = data[offset:]

	// count the number of backticks in the delimiter
	nb := skipChar(data, 0, '`')

	// find the next delimiter
	i, end := 0, 0
	hasLFBeforeDelimiter := false
	for end = nb; end < len(data) && i < nb; end++ {
		if data[end] == '\n' {
			hasLFBeforeDelimiter = true
		}
		if data[end] == '`' {
			i++
		} else {
			i = 0
		}
	}

	// no matching delimiter?
	if i < nb && end >= len(data) {
		return 0, nil
	}

	// If there are non-space chars after the ending delimiter and before a '\n',
	// flag that this is not a well formed fenced code block.
	hasCharsAfterDelimiter := false
	for j := end; j < len(data); j++ {
		if data[j] == '\n' {
			break
		}
		if !IsSpace(data[j]) {
			hasCharsAfterDelimiter = true
			break
		}
	}

	// trim outside whitespace
	fBegin := nb
	for fBegin < end && data[fBegin] == ' ' {
		fBegin++
	}

	fEnd := end - nb
	for fEnd > fBegin && data[fEnd-1] == ' ' {
		fEnd--
	}

	if fBegin == fEnd {
		return end, nil
	}

	// if delimiter has 3 backticks
	if nb == 3 {
		i := fBegin
		// If we found a '\n' before the end marker and there are only spaces
		// after the end marker, then this is a code block.
		if hasLFBeforeDelimiter && !hasCharsAfterDelimiter {
			codeblock := CodeBlock{}
			codeblock.Literal = data[i:fEnd]
			return end, codeblock
		}
	}

	// render the code span
	code := &Code{}
	code.Literal = data[fBegin:fEnd]
	return end, code
}

// newline preceded by two spaces becomes <br>
func maybeLineBreak(p *Parser, data []byte, offset int) (int, Node) {
	origOffset := offset
	offset = skipChar(data, offset, ' ')

	if offset < len(data) && data[offset] == '\n' {
		if offset-origOffset >= 2 {
			return offset - origOffset + 1, nil // &ast.Hardbreak{}
		}
		return offset - origOffset, nil
	}
	return 0, nil
}

type linkType int

const (
	linkNormal linkType = iota
	linkImg
)

func isReferenceStyleLink(data []byte, pos int, t linkType) bool {
	return pos < len(data)-1 && data[pos] == '[' && data[pos+1] != '^'
}

func maybeImage(p *Parser, data []byte, offset int) (int, Node) {
	if offset < len(data)-1 && data[offset+1] == '[' {
		return link(p, data, offset)
	}
	return 0, nil
}

// '[': parse a link or an image or a footnote or a citation
func link(p *Parser, data []byte, offset int) (int, Node) {
	// no links allowed inside regular links, footnote, and deferred footnotes
	if p.insideLink && (offset > 0 && data[offset-1] == '[' || len(data)-1 > offset && data[offset+1] == '^') {
		return 0, nil
	}

	var t linkType
	switch {
	// ![alt] == image
	case offset >= 0 && data[offset] == '!':
		t = linkImg
		offset++
	default:
		t = linkNormal
	}

	data = data[offset:]

	var (
		i                       = 1
		title, link, altContent []byte
		textHasNl               = false
		refDefContent           []byte // save markdown content from reference definition
	)

	// look for the matching closing bracket
	for level := 1; level > 0 && i < len(data); i++ {
		switch {
		case data[i] == '\n':
			textHasNl = true

		case data[i-1] == '\\':
			continue

		case data[i] == '[':
			level++

		case data[i] == ']':
			level--
			if level <= 0 {
				i-- // compensate for extra i++ in for loop
			}
		}
	}

	if i >= len(data) {
		return 0, nil
	}

	txtE := i
	i++

	// skip any amount of whitespace or newline
	// (this is much more lax than original markdown syntax)
	i = skipSpace(data, i)

	// inline style link
	switch {
	case i < len(data) && data[i] == '(':
		// skip initial whitespace
		i++

		i = skipSpace(data, i)

		linkB := i
		brace := 0

		// look for link end: ' " )
	findlinkend:
		for i < len(data) {
			switch {
			case data[i] == '\\':
				i += 2

			case data[i] == '(':
				brace++
				i++

			case data[i] == ')':
				if brace <= 0 {
					break findlinkend
				}
				brace--
				i++

			case data[i] == '\'' || data[i] == '"':
				break findlinkend

			default:
				i++
			}
		}

		if i >= len(data) {
			return 0, nil
		}
		linkE := i

		// look for title end if present
		titleB, titleE := 0, 0
		if data[i] == '\'' || data[i] == '"' {
			i++
			titleB = i
			titleEndCharFound := false

		findtitleend:
			for i < len(data) {
				switch {
				case data[i] == '\\':
					i++

				case data[i] == data[titleB-1]: // matching title delimiter
					titleEndCharFound = true

				case titleEndCharFound && data[i] == ')':
					break findtitleend
				}
				i++
			}

			if i >= len(data) {
				return 0, nil
			}

			// skip whitespace after title
			titleE = i - 1
			for titleE > titleB && IsSpace(data[titleE]) {
				titleE--
			}

			// check for closing quote presence
			if data[titleE] != '\'' && data[titleE] != '"' {
				titleB, titleE = 0, 0
				linkE = i
			}
		}

		// remove whitespace at the end of the link
		for linkE > linkB && IsSpace(data[linkE-1]) {
			linkE--
		}

		// remove optional angle brackets around the link
		if data[linkB] == '<' {
			linkB++
		}
		if data[linkE-1] == '>' {
			linkE--
		}

		// build escaped link and title
		if linkE > linkB {
			link = data[linkB:linkE]
		}

		if titleE > titleB {
			title = data[titleB:titleE]
		}

		i++

	// reference style link
	case isReferenceStyleLink(data, i, t):
		var id []byte

		// look for the id
		i++
		linkB := i
		i = skipUntilChar(data, i, ']')

		if i >= len(data) {
			return 0, nil
		}
		linkE := i

		// find the reference
		if linkB == linkE {
			if textHasNl {
				var b bytes.Buffer

				for j := 1; j < txtE; j++ {
					switch {
					case data[j] != '\n':
						b.WriteByte(data[j])
					case data[j-1] != ' ':
						b.WriteByte(' ')
					}
				}

				id = b.Bytes()
			} else {
				id = data[1:txtE]
			}
		} else {
			id = data[linkB:linkE]
		}

		// find the reference with matching id
		lr, ok := p.getRef(string(id))
		if !ok {
			return 0, nil
		}

		// keep link and title from reference
		link = lr.link
		title = lr.title
		refDefContent = lr.content
		i++

	// shortcut reference style link or reference or inline footnote
	default:
		var id []byte

		// craft the id
		if textHasNl {
			var b bytes.Buffer

			for j := 1; j < txtE; j++ {
				switch {
				case data[j] != '\n':
					b.WriteByte(data[j])
				case data[j-1] != ' ':
					b.WriteByte(' ')
				}
			}

			id = b.Bytes()
		} else {
			id = data[1:txtE]
		}

		// find the reference with matching id
		lr, ok := p.getRef(string(id))
		if !ok {
			return 0, nil
		}

		// keep link and title from reference
		link = lr.link
		// if inline footnote, title == footnote contents
		title = lr.title

		// rewind the whitespace
		i = txtE + 1
	}

	var uLink []byte
	if t == linkNormal || t == linkImg {
		if len(link) > 0 {
			var uLinkBuf bytes.Buffer
			unescapeText(&uLinkBuf, link)
			uLink = uLinkBuf.Bytes()
		}

		// links need something to click on and somewhere to go
		// [](http://bla) is legal in CommonMark, so allow txtE <=1 for linkNormal
		// [bla]() is also legal in CommonMark, so allow empty uLink
	}

	// Use the reference's definition md content. This is necessary for replacing in synchronizer,
	// as it doesn't care about the ref itself, but only about the file path
	// which is located in the definition block
	var content []byte
	if len(refDefContent) > 0 {
		content = refDefContent
	} else {
		content = data[:i]
	}

	// call the relevant rendering function
	switch t {
	case linkNormal:
		link := &Link{
			Destination: uLink,
			Title:       title,
			Leaf:        Leaf{Content: content},
		}
		if len(altContent) > 0 {
			p.AppendNode(newTextNode(altContent))
		} else {
			// links cannot contain other links, so turn off link parsing
			// temporarily and recurse
			insideLink := p.insideLink
			p.insideLink = true
			p.Inline(data[1:txtE])
			p.insideLink = insideLink
		}
		return i, link

	case linkImg:
		image := &Image{
			Destination: uLink,
			Title:       title,
			Leaf:        Leaf{Content: content},
		}
		p.AppendNode(newTextNode(data[1:txtE]))
		return i + 1, image

	default:
		return 0, nil
	}
}

func (p *Parser) inlineHTMLComment(data []byte) int {
	if len(data) < 5 {
		return 0
	}
	if data[0] != '<' || data[1] != '!' || data[2] != '-' || data[3] != '-' {
		return 0
	}
	i := 5
	// scan for an end-of-comment marker, across lines if necessary
	for i < len(data) && !(data[i-2] == '-' && data[i-1] == '-' && data[i] == '>') {
		i++
	}
	// no end-of-comment marker
	if i >= len(data) {
		return 0
	}
	return i + 1
}

// '<'
func leftAngle(p *Parser, data []byte, offset int) (int, Node) {
	data = data[offset:]

	end := tagLength(data)
	if size := p.inlineHTMLComment(data); size > 0 {
		end = size
	}
	if end <= 2 {
		return end, nil
	}
	htmlTag := &HTMLSpan{}
	htmlTag.Literal = data[:end]
	return end, htmlTag
}

// '\\' backslash escape
var escapeChars = []byte("\\`*_{}[]()#+-.!:|&<>~^")

func escape(p *Parser, data []byte, offset int) (int, Node) {
	data = data[offset:]

	if len(data) <= 1 {
		return 2, nil
	}

	if bytes.IndexByte(escapeChars, data[1]) < 0 {
		return 0, nil
	}

	return 2, newTextNode(data[1:2])
}

func unescapeText(ob *bytes.Buffer, src []byte) {
	i := 0
	for i < len(src) {
		org := i
		for i < len(src) && src[i] != '\\' {
			i++
		}

		if i > org {
			ob.Write(src[org:i])
		}

		if i+1 >= len(src) {
			break
		}

		ob.WriteByte(src[i+1])
		i += 2
	}
}

// return the length of the given tag, or 0 is it's not valid
func tagLength(data []byte) (end int) {
	var i int

	// a valid tag can't be shorter than 3 chars
	if len(data) < 3 {
		return 0
	}

	// begins with a '<' optionally followed by '/', followed by letter or number
	if data[0] != '<' {
		return 0
	}

	if data[1] == '/' {
		i = 2
	} else {
		i = 1
	}

	if !IsAlnum(data[i]) {
		return 0
	}

	i += bytes.IndexByte(data[i:], '>')
	if i < 0 {
		return 0
	}
	return i + 1
}

func newTextNode(d []byte) *Text {
	return &Text{Leaf: Leaf{Literal: d}}
}
