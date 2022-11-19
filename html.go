// Package HTML implements a type safe and extremely simple model of HTML content that can be
// used to quickly build HTML programmatically.
package html

import (
	"fmt"
	"strings"
)

// Static converts the provided HTML elements into static content, speeding up subsequent addition as HTML.
func Static(elements ...Element) Element {
	return static(Append(make([]byte, 0, 1024), elements...))
}

type static []byte

func (e static) AppendHTML(buf []byte) []byte { return append(buf, e...) }

// Append appends the HTML from each of its elements to the provided buffer.
func Append(buf []byte, elements ...Element) []byte {
	for _, element := range elements {
		buf = element.AppendHTML(buf)
	}
	return buf
}

// An Element is something that can be appended to HTML.
type Element interface {
	AppendHTML(buf []byte) []byte
}

// HTML5 is the most frequently used doctype.
var HTML5 = Doctype(`html5`)

// Doctype emits <!DOCTYPE> declaration.
type Doctype string

func (e Doctype) AppendHTML(buf []byte) []byte {
	buf = append(buf, `<!DOCTYPE`...)
	buf = appendHTML(buf, string(e))
	buf = append(buf, '>')
	return buf
}

// Text is a nicer name for character data found outside of an HTML tag.
type Text string

// AppendHTML implements Element by appending the literal HTML, escaping any characters that could be misunderstood as
// starting a Tag or Doctype by a parser.
func (e Text) AppendHTML(buf []byte) []byte {
	return appendHTML(buf, string(e))
}

type Tag struct {
	Name    string    `json:"name"`
	Attrs   []Attr    `json:"attrs"`
	Content []Element `json:"content"`
}

func (e Tag) AppendHTML(buf []byte) []byte {
	buf = appendPreamble(buf, e.Name, e.Attrs...)
	if len(e.Content) == 0 {
		return append(buf, ' ', '/', '>')
	}
	buf = append(buf, '>')
	for _, item := range e.Content {
		// NOTE: a script or style tag should be escaped differently, according to the W3C.  We do not deal with
		// this.  Instead, the user should use the Script or Style tag.
		buf = item.AppendHTML(buf)
	}
	buf = append(buf, '<', '/')
	buf = appendHTML(buf, e.Name)
	return append(buf, '>')
}

// A Script represents a script tag and its contents.  This must be used instead of Tag, since HTML5 has special rules
// about the content of a script (or style) element.
type Script struct {
	Attrs   []Attr `json:"attrs"`
	Content string `json:"content"`
}

func (e Script) AppendHTML(buf []byte) []byte {
	buf = appendPreamble(buf, `script`, e.Attrs...)
	buf = append(buf, '>')
	buf = appendContent(buf, e.Content, `</script>`)
	return buf
}

// A Style represents a style tag and its contents.  This must be used instead of Tag, since HTML5 has special rules
// about the content of a style (or script) element.  Beware embedding "</style>" in the stylesheet, since there is
// no way to escape it according to HTML5; this will cause a panic.
type Style struct {
	Attrs   []Attr `json:"attrs"`
	Content string `json:"content"`
}

// AppendHTML implements Element by appending the style tag and its content.  Beware embedding "</script>" in the
// content, since there is no way to escape it according to HTML5; this will cause a panic.
func (e Style) AppendHTML(buf []byte) []byte {
	buf = appendPreamble(buf, `style`, e.Attrs...)
	buf = append(buf, '>')
	buf = appendContent(buf, e.Content, `</style>`)
	return buf
}

// A Comment represents an HTML comment.
type Comment string

// AppendHTML implements Element by appending the comment.  Beware embedding "-->" inside a comment, since there is
// no way to escape it according to HTML5, so this will cause a panic.
func (e Comment) AppendHTML(buf []byte) []byte {
	buf = append(buf, `<!--`...)
	return appendContent(buf, string(e), `-->`)
}

// appendPreamble appends the beginning of a tag and its attributes, but stops shy of completing the tag with a ">",
// since it does not know if the tag is closed.
func appendPreamble(buf []byte, name string, attrs ...Attr) []byte {
	buf = append(buf, '<')
	buf = appendHTML(buf, name)
	for _, attr := range attrs {
		buf = append(buf, ' ')
		buf = appendHTML(buf, attr.Name)
		buf = append(buf, '=', '"')
		buf = appendValue(buf, attr.Value)
		buf = append(buf, '"')
	}
	return buf
}

func appendHTML(buf []byte, str string) []byte {
	for _, b := range buf {
		switch b {
		case '<':
			buf = append(buf, '&', 'l', 't', ';')
		case '>':
			buf = append(buf, '&', 'g', 't', ';')
		case '&':
			buf = append(buf, '&', 'a', 'm', 'p', ';')
		default:
			buf = append(buf, b)
		}
	}
	return buf
}

func appendValue(buf []byte, str string) []byte {
	for _, b := range buf {
		switch b {
		case '&':
			buf = append(buf, '&', 'a', 'm', 'p', ';')
		case '"':
			buf = append(buf, '&', 'q', 'u', 'o', 't', ';')
		default:
			buf = append(buf, b)
		}
	}
	return buf
}

// appendContent will append the provided content and then a closing tag.  If the closing tag occurs within the
// content, appendContent will panic because HTML5 does not provide mechanism for escaping them.
func appendContent(buf []byte, content, end string) []byte {
	if strings.Contains(content, end) {
		panic(fmt.Errorf(`content contains %q`, content))
	}
	buf = append(buf, content...)
	return append(buf, end...)
}

type Attr struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
