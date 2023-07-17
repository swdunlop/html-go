package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
)

var opt struct {
	Defer bool
}

func main() {
	flag.Usage = usage
	flag.BoolVar(&opt.Defer, `defer`, false, `use defer attribute for <script> tags`)
	flag.Parse()

	for _, path := range os.Args[1:] {
		dep, err := resolve(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "!! %v for %q\n", err, path)
		}
		fmt.Println(dep)
	}
}

func usage() {
	os.Stderr.WriteString(`USAGE: unpkg [-defer] <path>...
FLAGS:
  -defer  Use defer attribute for <script> tags

This utility queries unpkg.com for dependencies and follows redirects to the full URL then outputs a script or link tag
with SRI information and disabled referrer policy.
  
  unpkg alpinejs
  unpkg alpinejs@latest
  unpkg alpinejs@3.12.0 
  unpkg alpinejs/dist/cdn.min.js
  unpkg alpinejs@latest/dist/cdn.min.js
`)
}

// Unpkg outputs a tag with SRI for the provided path on unpkg for the given path.
// Example: unpkg react@latest react.min.js
func resolve(path string) (string, error) {
	pkg := strings.SplitN(path, `/`, 2)[0]
	meta, err := fetchUnpkgMeta(path)
	if err != nil {
		return ``, err
	}

	var template string
	switch meta.ContentType {
	case `application/javascript`:
		template = `<script defer src="$url" integrity="$integrity" crossorigin="anonymous" referrerpolicy="no-referrer"></script>`
	case `text/css`:
		template = `<link rel="stylesheet" href="$url" integrity="$integrity" crossorigin="anonymous" referrerpolicy="no-referrer">`
	default:
		return ``, fmt.Errorf(`unknown content type %q`, meta.ContentType)
	}
	return expandHTML(template, map[string]string{
		`pkg`:       pkg,
		`path`:      meta.Path,
		`integrity`: meta.Integrity,
		`url`:       strings.TrimSuffix(meta.URL, `?meta`),
	})
}

func fetchUnpkgMeta(path string) (*unpkgMeta, error) {
	var err error
	meta := new(unpkgMeta)
	meta.URL, err = getJSON(meta, `https://unpkg.com/`+path+`?meta`)
	if err != nil {
		return nil, err
	}
	return meta, nil
}

type unpkgMeta struct {
	URL          string `json:"url"`
	Path         string `json:"path"`
	Type         string `json:"type"`
	ContentType  string `json:"contentType"`
	Integrity    string `json:"integrity"`
	LastModified string `json:"lastModified"`
	Size         int64  `json:"size"`
}

// expandHTML expands the template the replaces the following in its expansions with HTML entities: '&', '"', and '<'
func expandHTML(template string, table map[string]string) (string, error) {
	var err error
	result := os.Expand(template, func(param string) string {
		if err != nil {
			return param
		}
		str, ok := table[param]
		if !ok {
			return param
		}

		return entityReplacer.Replace(str)
	})

	return result, err
}

var entityReplacer = strings.NewReplacer(
	`&`, `&amp;`,
	`"`, `&quot;`,
	`<`, `&lt;`,
)

func getJSON(v any, url string) (string, error) {
	req, err := http.NewRequest(`GET`, url, nil)
	if err != nil {
		return url, err
	}
	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return url, err
	}
	defer func() { _ = rsp.Body.Close() }()
	url = rsp.Request.URL.String()
	switch rsp.StatusCode {
	case 200:
		err = json.NewDecoder(rsp.Body).Decode(v)
		return url, err
	default:
		return url, fmt.Errorf(`%v while fetching %v`, rsp.Status, url)
	}
}
