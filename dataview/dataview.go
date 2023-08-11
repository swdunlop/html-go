// Package dataview provides a way to view Go values as tabular HTML if the values can be represented as JSON.
package dataview

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"

	"github.com/swdunlop/html-go"
	"github.com/tidwall/gjson"
)

// Stylesheet will return the structural CSS needed to render the dataview.  The options are currently ignored, but are
// present in case we need to add options like a class prefix in the future.
func Stylesheet(options ...Option) string {
	return stylesheet
}

const stylesheet = `
.object, .array, .table { display: grid; width: fit-content; }
.row { display: contents; }
.object { grid-template-columns: minmax(min-content, max-content) 1fr; }
`

// From converts a Go value into HTML, converting it into JSON first and parsing it with GJSON.
func From(data any, options ...Option) html.Content {
	js, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	return FromJSON(js, options...)
}

// FromJSON converts a JSON document into HTML, parsing it with GJSON -- the provided JSON MUST be valid.
func FromJSON(js []byte, options ...Option) html.Content {
	return FromGJSON(gjson.ParseBytes(js), options...)
}

// FromGJSON converts a GJSON result into HTML.  This is the most efficient way to use dataview if you already
// have a GJSON result.
func FromGJSON(data gjson.Result, options ...Option) html.Content {
	cfg := &config{}
	for _, option := range options {
		option(cfg)
	}
	return cfg.asContent(data, ``)
}

// Hook registers a function that replaces how a value is rendered if the path to the value matches the provided
// pattern.
//
// Patterns are regex patterns that match paths like .persons.0.name or .persons.0.address.city.
// If a hook returns nil, the default rendering is used.
func Hook(rx *regexp.Regexp, hookFn func(path string, data gjson.Result) html.Content) Option {
	return func(cfg *config) {
		cfg.hooks = append(cfg.hooks, hook{rx, hookFn})
	}
}

// TableHook registers a function that converts an array containing at least one object into some other GJSON result
// if the path to the array matches the provided pattern.  TableHooks are applied before Hooks and use the same path
// syntax.
func TableHook(rx *regexp.Regexp, hookFn func(path string, data gjson.Result) gjson.Result) Option {
	return func(cfg *config) {
		cfg.tableHooks = append(cfg.tableHooks, tableHook{rx, hookFn})
	}
}

type Option func(*config)

type config struct {
	hooks      []hook
	tableHooks []tableHook
}

type hook struct {
	rx   *regexp.Regexp
	hook func(path string, data gjson.Result) html.Content
}

type tableHook struct {
	rx   *regexp.Regexp
	hook func(path string, data gjson.Result) gjson.Result
}

func (cfg *config) asContent(data gjson.Result, path string) html.Content {
	if isTabular(data) {
		for _, hook := range cfg.tableHooks {
			if hook.rx.MatchString(path) {
				data = hook.hook(path, data)
				if !isTabular(data) {
					goto notTabular
				}
			}
		}
		for _, hook := range cfg.hooks {
			if hook.rx.MatchString(path) {
				content := hook.hook(path, data)
				if content != nil {
					return content
				}
			}
		}
		return cfg.tableAsContent(data, path)
	}

notTabular:
	for _, hook := range cfg.hooks {
		if hook.rx.MatchString(path) {
			content := hook.hook(path, data)
			if content != nil {
				return content
			}
		}
	}
	return cfg.render(data, path)
}

func (cfg *config) render(data gjson.Result, path string) html.Content {
	switch data.Type {
	case gjson.Null:
		return html.HTML(`<span class='null'>null</span>`)
	case gjson.False:
		return html.HTML(`<span class='bool'>false</span>`)
	case gjson.True:
		return html.HTML(`<span class='bool'>true</span>`)
	case gjson.Number:
		if len(data.Raw) > 0 {
			return html.HTML(data.Raw)
		}
		return html.HTML(data.String())
	case gjson.String:
		return html.Text(data.String()) // TODO: wrap in a span to enable ellipsis?
	default:
		switch {
		case data.IsArray():
			return cfg.arrayAsContent(data, path)
		case data.IsObject():
			return cfg.objectAsContent(data, path)
		default:
			panic(fmt.Errorf(`unknown gjson type %v at %q`, data.Type, path))
		}
	}
}

func (cfg *config) arrayAsContent(data gjson.Result, path string) html.Content {
	seq := data.Array()
	n := len(seq)
	if n == 0 {
		return html.HTML(`<div class='array empty'>[]</div>`)
	}
	// TODO: visualization for empty arrays.
	table := make(html.Group, 0, n+4) // header, body, footer
	table = append(table, html.HTML(`<div class='array'>`))
	path += "."
	for ix, value := range seq {
		table = append(table, html.Group{
			html.HTML(`<div class='value'>`),
			cfg.asContent(value, path+strconv.Itoa(ix)),
			html.HTML(`</div>`),
		})
		ix++
	}
	table = append(table, html.HTML(`</div>`))
	return table
}

func (cfg *config) tableAsContent(data gjson.Result, path string) html.Content {
	seq := data.Array()
	// We do two passes, one to identify all of the keys of any embedded objects, and another to build a table where
	// each item has a row.
	//
	// If there are no embedded objects, we show a single column table with no heading.
	// Otherwise, we show a table with one column per key, with a heading row.
	//
	// This must tolerate mixtures of objects and slices or literals.
	var columns = struct {
		labels []string
		index  map[string]int
	}{
		make([]string, 0, 32),
		make(map[string]int, 32),
	}

	for _, value := range seq {
		if value.IsObject() {
			value.ForEach(func(key, _ gjson.Result) bool {
				if _, ok := columns.index[key.Str]; !ok {
					columns.index[key.Str] = len(columns.labels)
					columns.labels = append(columns.labels, key.Str)
				}
				return true
			})
		}
	}

	table := make(html.Group, 0, len(columns.labels)*3+len(seq)+2)
	table = append(table, html.HTML(fmt.Sprint(
		`<div class='table' style='grid-template-columns: repeat(`,
		len(columns.labels),
		`, minmax(min-content, max-content));'>`,
	)))
	for _, label := range columns.labels {
		table = append(table, html.Group{
			html.HTML(`<div class='header label'>`),
			html.Text(label),
			html.HTML(`</div>`),
		})
	}
	path += "."
	for ix, value := range seq {
		table = append(table, html.HTML(`<div class='row'>`))
		if value.IsObject() {
			row := make(html.Group, 0, len(columns.labels)*3)
			for _, label := range columns.labels {
				data := value.Get(label)
				if data.Exists() {
					row = append(row,
						html.HTML(`<div class='value'>`),
						// html.Text(label),
						// html.HTML(`'>`), //TODO: add class for label
						cfg.asContent(data, path+label),
						html.HTML(`</div>`),
					)
				} else {
					row = append(row, html.HTML(`<div class='value na'>N/A</div>`))
				}
			}
			table = append(table, row)
		} else {
			table = append(table, html.Group{
				html.HTML(`<div class='value' style='grid-column: 1/-1;'>`), // full width
				cfg.asContent(value, path+strconv.Itoa(ix)),
				html.HTML(`</div>`),
			})
		}
		table = append(table, html.HTML(`</div>`))
	}

	table = append(table, html.HTML(`</div>`))
	return table
}

func (cfg *config) objectAsContent(data gjson.Result, path string) html.Content {
	// We show objects as a table with two columns, one for the keys, and one for the values.
	table := make(html.Group, 0, data.Get(`#`).Int())
	table = append(table, html.HTML(`<div class='object'>`))
	path += "."
	data.ForEach(func(key, value gjson.Result) bool {
		table = append(table, html.Group{
			html.HTML(`<div class='key label'>`),
			html.Text(key.Str),
			html.HTML(`</div><div class='value'>`),
			cfg.asContent(value, path+key.Str),
			html.HTML(`</div>`),
		})
		return true
	})
	return append(table, html.HTML(`</div>`))
}

func isTabular(data gjson.Result) bool {
	if !data.IsArray() {
		return false
	}
	tabular := false
	data.ForEach(func(_, value gjson.Result) bool {
		if value.IsObject() {
			tabular = true
			return false
		}
		return true
	})
	return tabular
}
