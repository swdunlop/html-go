// Package tag provides an interface for incrementally building HTML tags with content, taking a trick from the "m"
// function in MithrilJS and running with it in a direction that may be familiar to people who have used Zerolog.
package tag

import (
	"fmt"
	"strings"

	"github.com/swdunlop/html-go"
)

// New will construct a new HTML tag using the provided string like a CSS selector -- parsing out the tag name,
// ID and any classes.  If a name is omitted, it is assumed to be "div".  If an ID or Class is omitted, it is
// omitted from the produced tag.
//
// Attributes can also be specified using a CSS-like syntax appending "[name]" for boolean attributes or "[name=value]"
// for attributes with values.  Multiple attributes can be added this way, wrapping each one in square brackets.
//
// Content can be added to the tag by passing it as variadic arguments to New.  If the tag is a "void" tag, like "link",
// then it cannot actually have any content.  Instead, the additional content will be appended after the tag.
func New(selector string, content ...html.Content) Interface {
	var t tag
	t.parseSelector(selector)
	t.content = extend(t.content, content...)
	return t
}

// Interface describes the interface returned by tag.New and various methods of this interface.  In general, each
// time a method could change the value of this interface, a copy is returned.
type Interface interface {
	// AppendHTML implements html.Content by appending the tag and its content to the buffer.
	AppendHTML(buf []byte) []byte

	// Class will append classes to the tag, but not remove the previous classes.  If you want to reset the set
	// of classes, use the "Attribute" method.
	Class(classes ...string) Interface

	// Set will return a copy of the tag with additional attributes.  If the attribute was already set, the previous
	// value will be removed.  If no values are provided, a "boolean" attribute is added, like the "defer" attribute of
	// script.
	Set(attribute string, values ...any) Interface

	// Add will return a copy of the tag with additional content.  If the tag is a "void" tag, like "link", then it
	// cannot actually have any content.  Instead, the additional content will be appended after the tag.
	//
	// If the tag is a "style" or "script", Add will fail unless the content is html.Text -- you can only add text
	// due to HTML5 rules.
	Add(content ...html.Content) Interface

	// Text will use fmt.Sprint to coerce data into text and add it as HTML content to the tag.
	Text(data ...any) Interface
}

type tag struct {
	name       string
	classes    []string
	attributes []attribute
	content    []html.Content
	void       bool
}

func (t *tag) parseSelector(src string) {
	defer t.determineVoid()
	var id, class string
	t.name = `div`
	if src == "" {
		return
	}
	pos := strings.IndexAny(src, "#.[")
	if pos == -1 {
		t.name = src
		return
	}
	if pos > 0 {
		t.name = src[:pos]
	}
	t.attributes = make([]attribute, 1) // preallocate space for id
	src = src[pos:]
	for len(src) > 0 {
		switch src[0] {
		case '#':
			pos = strings.IndexAny(src[1:], ".[") + 1
			if pos == 0 {
				pos = len(src)
			}
			// TODO: reject invalid id names
			id = src[1:pos]
		case '.':
			pos = strings.IndexAny(src[1:], "#.[") + 1
			if pos == 0 {
				pos = len(src)
			}
			if class != "" {
				class += ` `
			}
			// TODO: reject invalid class names
			class += src[1:pos]
		case '[':
			pos = strings.IndexByte(src[1:], ']') + 1
			if pos == 0 {
				panic(fmt.Errorf(`%q starts with [ but does not end with ]`, src))
			}
			if pos > 1 {
				t.addLiteralAttribute(src[1:pos])
			} else {
				// ignore empty attributes, arguably we should panic.
			}
			pos++
		}
		src = src[pos:]
	}
	if class != "" {
		t.classes = []string{class}
	}
	if id != "" {
		t.attributes[0] = attribute{`id`, id}
	} else {
		t.attributes = t.attributes[1:]
	}
	return
}

func (t *tag) addLiteralAttribute(src string) {
	ix := strings.IndexByte(src, '=')
	if ix < 0 {
		// assume a boolean attribute.
		t.attributes = append(t.attributes, attribute{head: src})
		return
	}
	attribute := attribute{head: src[:ix]}
	tail := src[ix+1:]
	buf := make([]byte, 0, len(tail)+2)
	buf = appendValueStr(buf, tail)
	attribute.tail = string(buf)
	// append is safe here because we preallocate space for the attributes.
	t.attributes = append(t.attributes, attribute)
}

func (t *tag) determineVoid() {
	switch t.name {
	case `area`, `base`, `br`, `col`, `embed`, `hr`, `img`, `input`, `keygen`, `link`, `meta`, `param`, `source`,
		`track`, `wbr`:
		t.void = true
	}
}

func (t tag) AppendHTML(buf []byte) []byte {
	buf = append(buf, '<')
	buf = append(buf, t.name...)
	if len(t.classes) > 0 {
		buf = append(buf, ` class='`...)
		buf = html.AppendText(buf, t.classes[0])
		for _, class := range t.classes[1:] {
			buf = append(buf, ' ')
			buf = html.AppendText(buf, class)
		}
		buf = append(buf, '\'')
	}
	for _, attr := range t.attributes {
		buf = append(buf, ' ')
		buf = append(buf, attr.head...)
		if len(attr.tail) > 0 {
			buf = append(buf, '=', '\'')
			buf = append(buf, attr.tail...)
			buf = append(buf, '\'')
		}
	}
	buf = append(buf, '>')
	for _, content := range t.content {
		buf = content.AppendHTML(buf)
	}
	if t.void {
		return buf
	}
	buf = append(buf, '<', '/')
	buf = append(buf, t.name...)
	buf = append(buf, '>')
	return buf
}

func (t tag) Class(classes ...string) Interface {
	t.classes = extend(t.classes, classes...)
	return t
}

func (t tag) Set(head string, values ...any) Interface {
	tail := make([]byte, 0, 64)
	if ix := strings.IndexByte(head, '='); ix > -1 {
		tail = appendValueStr(tail, head[ix+1:])
		head = head[:ix]
	}
	for _, value := range values {
		tail = appendValue(tail, value)
	}
	if head == `class` {
		// as a special case, if class is set, we replace the existing classes
		t.classes = []string{string(tail)}
		return t
	}
	for i := range t.attributes {
		if t.attributes[i].head == head {
			// copy the attributes so we do not modify the original
			t.attributes = append([]attribute(nil), t.attributes...)
			t.attributes[i].tail = string(tail)
			return t
		}
	}
	t.attributes = extend(t.attributes, attribute{head: head, tail: string(tail)})
	return t
}

func (t tag) Text(data ...any) Interface { return t.Add(html.Text(fmt.Sprint(data...))) }

func (t tag) Add(content ...html.Content) Interface {
	t.content = extend(t.content, content...)
	return t
}

type attribute struct {
	head string
	tail string
}

func fmtValue(values ...any) string {
	if len(values) == 0 {
		return ""
	}
	buf := make([]byte, 1, 64)
	buf[0] = '\''
	for _, value := range values {
		buf = appendValue(buf, value)
	}
	buf = append(buf, '\'')
	return string(buf)
}

func appendValue(buf []byte, value any) []byte       { return appendValueStr(buf, fmt.Sprint(value)) }
func appendValueStr(buf []byte, value string) []byte { return appendValueBytes(buf, []byte(value)) }
func appendValueBytes(buf []byte, value []byte) []byte {
	for _, ch := range value {
		switch ch {
		case '\'':
			buf = append(buf, `&apos;`...)
		case '&':
			buf = append(buf, `&amp;`...)
		default:
			buf = append(buf, ch)
		}
	}
	return buf
}

// extend is a helper function to extend a slice without using the capacity of the slice.
func extend[T any](slice []T, values ...T) []T {
	ret := make([]T, len(slice), len(slice)+len(values))
	copy(ret, slice)
	return append(ret, values...)
}
