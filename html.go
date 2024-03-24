// Package HTML implements very simple model of HTML content that is used to build HTML programmatically.
package html

// Map will apply a function to each item in the slice to return content.
func Map[S ~[]E, E any](slice S, fn func(E) Content) Group {
	result := make(Group, len(slice))
	for i, item := range slice {
		result[i] = fn(item)
	}
	return result
}

// Group is a slice of Content that can be appended as content.
type Group []Content

// AppendHTML appends the HTML from each of its elements to the provided buffer.
func (group Group) AppendHTML(buf []byte) []byte {
	for _, element := range group {
		buf = element.AppendHTML(buf)
	}
	return buf
}

// Static merges the provided HTML content into static content, speeding up subsequent addition as HTML.
func Static(elements ...Content) Content {
	return HTML(Append(make([]byte, 0, 1024), elements...))
}

// HTML is static HTML content.
type HTML []byte

// AppendHTML simply appends content to the HTML buffer without transformation.
func (html HTML) AppendHTML(buf []byte) []byte { return append(buf, html...) }

// Append appends the HTML from each of its elements to the provided buffer.
func Append(buf []byte, elements ...Content) []byte {
	for _, element := range elements {
		buf = element.AppendHTML(buf)
	}
	return buf
}

// A Content is something that can be appended as HTML in UTF-8 encoding to a HTML document.
type Content interface {
	AppendHTML(buf []byte) []byte
}

// Text is content that escapes the following characters using entities: "<", ">", "&", ";", "'" and '"'
type Text string

// AppendHTML implements Element by appending the literal HTML, escaping any characters that could be misunderstood as
// starting a Tag or Doctype by a parser.
func (text Text) AppendHTML(buf []byte) []byte { return AppendText(buf, string(text)) }

// AppendText appends literal text, escaping the following characters using entities:  "<", ">", "&", ";", "'" and '"'
func AppendText(buf []byte, text string) []byte {
	// We assume that the provided text is sound UTF-8 already, so really we are just looking for certain runes
	// that need to be expressed as entities to avoid XSS.
	for _, ch := range []byte(text) {
		switch ch {
		case '<':
			buf = append(buf, '&', 'l', 't', ';')
		case '>':
			buf = append(buf, '&', 'g', 't', ';')
		case '"':
			buf = append(buf, '&', 'q', 'u', 'o', 't', ';')
		case '\'':
			buf = append(buf, '&', 'a', 'p', 'o', 's', ';')
		case '&':
			buf = append(buf, '&', 'a', 'm', 'p', ';')
		default:
			buf = append(buf, ch)
		}
	}
	return buf
}

// A Func is content that only generates content when needed.
type Func func() Content

// AppendHTML implements Content by calling the function to get the content.
func (fn Func) AppendHTML(buf []byte) []byte { return fn().AppendHTML(buf) }
