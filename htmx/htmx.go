// Package htmx adds helper functions for HTMX applications.
package htmx

import (
	"net/http"

	"github.com/swdunlop/html-go"
	"github.com/swdunlop/html-go/alpine-ajax"
)

// RenderPage renders a full page if the HX-Target header is not present, otherwise, it uses Render to
// render the targeted part of the page.
func RenderPage(r *http.Request, page func(PartMap) html.Content, parts ...Part) html.Content {
	h := r.Header.Get(`HX-Target`)
	if h != `` {
		return render(h, parts...)
	}
	table := make(PartMap, len(parts))
	for _, part := range parts {
		table[part.ID()] = part
	}
	return page(table)
}

// Render parses the HX-Target header and returns the part that matches.  This will return an empty html.Group if no
// parts match.
//
// Parts with an empty ID will not be included in the output.
func Render(r *http.Request, parts ...Part) html.Content {
	target := r.Header.Get(`HX-Target`)
	return render(target, parts...)
}

func render(target string, parts ...Part) html.Content {
	for _, part := range parts {
		if part.ID() == target {
			return part
		}
	}
	return html.Group{}
}

// The Part interface describes a part of a page with an ID that can be requested by an HTMX client.  This
// interface is implemented by the tag.New function.
type Part interface {
	html.Content // Each part is HTML content.

	// ID returns the ID of the part, which can be used as a target in the HX-Target header.
	ID() string
}

// PartMap is a map of parts by ID provided to a page function by RenderPage.
type PartMap = alpine.PartMap
