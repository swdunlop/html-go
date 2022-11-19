// Package tag provides a system of functional options that build HTML tags programmatically.
package tag

import (
	"strings"

	"github.com/swdunlop/html-go"
)

// New constructs a new HTML tag and applies the provided options.
func New(name string, options ...Option) html.Tag {
	tag := html.Tag{Name: name}
	for _, option := range options {
		option(&tag)
	}
	return tag
}

// Factory constructs an html tag factory using functional options.  This is generally used to stamp out basic
// HTML element functions, applying some basic options as a template.
func Factory(name string, options ...Option) func(...Option) html.Tag {
	base := html.Tag{Name: name}
	for _, option := range options {
		option(&base)
	}
	return func(options ...Option) html.Tag {
		tag := base
		for _, option := range options {
			option(&tag)
		}
		return tag
	}
}

// Static appends static content inside the tag.
func Static(contents ...html.Element) Option {
	static := html.Static(contents...)
	return Content(static)
}

// Text appends text content inside the tag.
func Text(text string) Option {
	return Content(html.Text(text))
}

// Content appends content inside the tag.
func Content(contents ...html.Element) Option {
	return func(tag *html.Tag) {
		tag.Content = append(tag.Content, contents...)
	}
}

// ID appends an id attribute on the tag.
func ID(id string) Option {
	return Attr(`id`, id)
}

// Class appends a class attribute on the tag.
func Class(classes ...string) Option {
	return Attr(`class`, strings.Join(classes, ` `))
}

// Attr appends an attribute on the tag.
func Attr(name, value string) Option {
	return func(tag *html.Tag) {
		tag.Attrs = append(tag.Attrs, html.Attr{Name: name, Value: value})
	}
}

// Apply applies a series of options as an option.
func Apply(options ...Option) Option {
	return func(tag *html.Tag) {
		for _, option := range options {
			option(tag)
		}
	}
}

// An Option affects an HTML tag.
type Option func(*html.Tag)
