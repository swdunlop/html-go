package main

import (
	"io"
	"os"
	"regexp"

	"github.com/swdunlop/html-go"
	"github.com/swdunlop/html-go/dataview"
	"github.com/tidwall/gjson"
)

func main() {
	// Given a JSONL stream from stdin and an optional list of GJSON selectors, output a HTML document that
	// renders resulting JSON values as a series of dataviews.

	js, err := io.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}
	data := gjson.ParseBytes(js)

	doc := html.Append(make([]byte, 0, 1024*1024), html.Group{
		html.HTML(`<!DOCTYPE html><html><head><meta charset="utf-8">`),
		html.HTML(`<style>`),
		html.HTML(css),
		html.HTML(`</style></head><body>`),
		dataview.FromGJSON(data, dataview.Hook(rxOData, func(path string, data gjson.Result) html.Content {
			// We ellide @odata fields as uninteresting to the user.
			return ellide
		})),
		html.HTML(`</body></html>`),
	})

	_, err = os.Stdout.Write(doc)
	if err != nil {
		panic(err)
	}
}

var rxOData = regexp.MustCompile(`(?:^|\.)@odata[\.|$]`)
var ellide = html.HTML(`<div class="ellide">…</div>`)

// css extends the structural CSS from the dataview with colors, fonts and spacing.
var css = dataview.Stylesheet() + `
body{ background-color: #111; color: #eee; font-family: sans-serif; }
.object, .array, .table { border-top: 2px solid #888; }
.label { font-weight: bold; background-color: #333; }
.empty, .undefined, .null { font-style: italic; }
.label, .value { font-family: monospace; padding: .35em; }
`
