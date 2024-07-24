package parser

import (
	"bytes"
)

// Parse Block-level data.
// Note: this function and many that it calls assume that
// the input buffer ends with a newline.
func (p *Parser) Block(data []byte) {
	for len(data) > 0 {
		// blank lines.  note: returns the # of bytes to skip
		if i := IsEmpty(data); i > 0 {
			data = data[i:]
			continue
		}

		// indented code block:
		//
		//     func max(a, b int) int {
		//         if a > b {
		//             return a
		//         }
		//         return b
		//      }
		if p.codePrefix(data) > 0 {
			data = data[p.code(data):]
			continue
		}

		// fenced code block:
		//
		// ``` go
		// func fact(n int) int {
		//     if n <= 1 {
		//         return n
		//     }
		//     return n * fact(n-1)
		// }
		// ```
		if i := p.fencedCodeBlock(data); i > 0 {
			data = data[i:]
			continue
		}

		// anything else must look like a normal paragraph
		// note: this finds underlined headings, too
		idx := p.paragraph(data)
		data = data[idx:]
	}
}

func (p *Parser) AddBlock(b Container) Container {
	p.Blocks = append(p.Blocks, b)
	return b
}

func IsEmpty(data []byte) int {
	// it is okay to call isEmpty on an empty buffer
	if len(data) == 0 {
		return 0
	}

	var i int
	for i = 0; i < len(data) && data[i] != '\n'; i++ {
		if data[i] != ' ' && data[i] != '\t' {
			return 0
		}
	}
	i = skipCharN(data, i, '\n', 1)
	return i
}

// isFenceLine checks if there's a fence line (e.g., ``` or ``` go) at the beginning of data,
// and returns the end index if so, or 0 otherwise. It also returns the marker found.
// If syntax is not nil, it gets set to the syntax specified in the fence line.
func isFenceLine(data []byte, syntax *string, oldmarker string) (end int, marker string) {
	i, size := 0, 0

	n := len(data)
	// skip up to three spaces
	for i < n && i < 3 && data[i] == ' ' {
		i++
	}

	// check for the marker characters: ~ or `
	if i >= n {
		return 0, ""
	}
	if data[i] != '~' && data[i] != '`' {
		return 0, ""
	}

	c := data[i]

	// the whole line must be the same char or whitespace
	for i < n && data[i] == c {
		size++
		i++
	}

	// the marker char must occur at least 3 times
	if size < 3 {
		return 0, ""
	}
	marker = string(data[i-size : i])

	// if this is the end marker, it must match the beginning marker
	if oldmarker != "" && marker != oldmarker {
		return 0, ""
	}

	// if just read the beginning marker, read the syntax
	if oldmarker == "" {
		i = skipChar(data, i, ' ')
		if i >= n {
			if i == n {
				return i, marker
			}
			return 0, ""
		}

		syntaxStart, syntaxLen := syntaxRange(data, &i)
		if syntaxStart == 0 && syntaxLen == 0 {
			return 0, ""
		}

		// caller wants the syntax
		if syntax != nil {
			*syntax = string(data[syntaxStart : syntaxStart+syntaxLen])
		}
	}

	i = skipChar(data, i, ' ')
	if i >= n || data[i] != '\n' {
		if i == n {
			return i, marker
		}
		return 0, ""
	}
	return i + 1, marker // Take newline into account.
}

func syntaxRange(data []byte, iout *int) (int, int) {
	n := len(data)
	syn := 0
	i := *iout
	syntaxStart := i
	if data[i] == '{' {
		i++
		syntaxStart++

		for i < n && data[i] != '}' && data[i] != '\n' {
			syn++
			i++
		}

		if i >= n || data[i] != '}' {
			return 0, 0
		}

		// strip all whitespace at the beginning and the end
		// of the {} block
		for syn > 0 && IsSpace(data[syntaxStart]) {
			syntaxStart++
			syn--
		}

		for syn > 0 && IsSpace(data[syntaxStart+syn-1]) {
			syn--
		}

		i++
	} else {
		for i < n && !IsSpace(data[i]) {
			syn++
			i++
		}
	}

	*iout = i
	return syntaxStart, syn
}

// fencedCodeBlock returns the end index if data contains a fenced code block at the beginning,
func (p *Parser) fencedCodeBlock(data []byte) int {
	var syntax string
	beg, marker := isFenceLine(data, &syntax, "")
	if beg == 0 || beg >= len(data) {
		return 0
	}

	for {
		// safe to assume beg < len(data)

		// check for the end of the code block
		fenceEnd, _ := isFenceLine(data[beg:], nil, marker)
		if fenceEnd != 0 {
			beg += fenceEnd
			break
		}

		// copy the current line
		end := skipUntilChar(data, beg, '\n') + 1

		// did we reach the end of the buffer without a closing marker?
		if end >= len(data) {
			return 0
		}
		beg = end
	}
	return beg
}

// returns prefix length for block code
func (p *Parser) codePrefix(data []byte) int {
	n := len(data)
	if n >= 1 && data[0] == '\t' {
		return 1
	}
	if n >= 4 && data[3] == ' ' && data[2] == ' ' && data[1] == ' ' && data[0] == ' ' {
		return 4
	}
	return 0
}

func (p *Parser) code(data []byte) int {
	i := 0
	for i < len(data) {
		beg := i

		i = skipUntilChar(data, i, '\n')
		i = skipCharN(data, i, '\n', 1)

		blankline := IsEmpty(data[beg:i]) > 0
		if pre := p.codePrefix(data[beg:i]); pre > 0 {
			// beg += pre
		} else if !blankline {
			// non-empty, non-prefixed line breaks the pre
			i = beg
			break
		}
	}
	return i
}

// render a single paragraph that has already been parsed out
func (p *Parser) renderParagraph(data []byte) {
	if len(data) == 0 {
		return
	}

	// trim leading spaces
	beg := skipChar(data, 0, ' ')

	end := len(data)
	// trim trailing newline
	if data[len(data)-1] == '\n' {
		end--
	}

	// trim trailing spaces
	for end > beg && data[end-1] == ' ' {
		end--
	}
	para := &Paragraph{}
	para.Content = data[beg:end]
	p.AddBlock(para)
}

func (p *Parser) paragraph(data []byte) int {
	// i: index of cursor/end of current line
	var i int
	tabSize := tabSizeDefault
	// keep going until we find something to mark the end of the paragraph
	for i < len(data) {
		// mark the beginning of the current line
		current := data[i:]

		// did we find a reference or a footnote? If so, end a paragraph
		// preceding it and report that we have consumed up to the end of that
		// reference:
		if refEnd := isReference(p, current, tabSize); refEnd > 0 {
			p.renderParagraph(data[:i])
			return i + refEnd
		}

		// did we find a blank line marking the end of the paragraph?
		if n := IsEmpty(current); n > 0 {
			p.renderParagraph(data[:i])
			return i + n
		}

		// if there's a fenced code block, paragraph is over
		if p.fencedCodeBlock(current) > 0 {
			p.renderParagraph(data[:i])
			return i
		}

		// otherwise, scan to the beginning of the next line
		nl := bytes.IndexByte(data[i:], '\n')
		if nl >= 0 {
			i += nl + 1
		} else {
			i += len(data[i:])
		}
	}

	p.renderParagraph(data[:i])
	return i
}

// skipChar advances i as long as data[i] == c
func skipChar(data []byte, i int, c byte) int {
	n := len(data)
	for i < n && data[i] == c {
		i++
	}
	return i
}

// like skipChar but only skips up to max characters
func skipCharN(data []byte, i int, c byte, max int) int {
	n := len(data)
	for i < n && max > 0 && data[i] == c {
		i++
		max--
	}
	return i
}

// skipUntilChar advances i as long as data[i] != c
func skipUntilChar(data []byte, i int, c byte) int {
	n := len(data)
	for i < n && data[i] != c {
		i++
	}
	return i
}

func skipSpace(data []byte, i int) int {
	n := len(data)
	for i < n && IsSpace(data[i]) {
		i++
	}
	return i
}
