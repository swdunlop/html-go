// Package alpine adds helper functions for Alpine AJAX applications.
package alpine

import (
	"net/http"
	"strings"

	"github.com/swdunlop/html-go"
)

// RenderPage renders a full page if the X-Alpine-Target header is not present, otherwise, it uses Render to
// render the requested parts of the page.
func RenderPage(r *http.Request, page func(PartMap) html.Content, parts ...Part) html.Content {
	h := r.Header.Get(`X-Alpine-Target`)
	if h != `` {
		return render(h, parts...)
	}
	table := make(PartMap, len(parts))
	for _, part := range parts {
		table[part.ID()] = part
	}
	return page(table)
}

// Render parses the X-Alpine-Target header and returns a html.Group containing only the requested parts, in the order
// they were specified as arguments to render.  (This is specified so the last part can be an "errors" part that lists
// any errors that occurred while rendering the other parts.)
//
// Parts with an empty ID will not be included in the output.
func Render(r *http.Request, parts ...Part) html.Group {
	h := r.Header.Get(`X-Alpine-Target`)
	return render(h, parts...)
}

func render(h string, parts ...Part) html.Group {
	seq := strings.Split(h, ` `)
	targets := make(map[string]struct{}, len(seq))
	for _, target := range seq {
		if target == `` {
			continue
		}
		targets[target] = struct{}{}
	}
	group := make(html.Group, 0, len(parts))
	for _, part := range parts {
		id := part.ID()
		if _, ok := targets[id]; !ok {
			group = append(group, part)
		}
	}
	return group
}

// The Part interface describes a part of a page with an ID that can be requested by an Alpine AJAX client.  This
// interface is implemented by the tag.New function.
type Part interface {
	html.Content // Each part is HTML content.

	// ID returns the ID of the part, which can be used as a target in the X-Alpine-Target header.
	ID() string
}

// PartMap is a map of parts by ID provided to a page function by RenderPage.
type PartMap map[string]Part

// Get returns the content for the part with the given ID, or empty content if no such part exists.
func (table PartMap) Get(id string) html.Content {
	content, ok := table[id]
	if !ok {
		return html.Group{} // no content
	}
	return content
}
