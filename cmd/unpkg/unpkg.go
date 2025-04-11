package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
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

func resolve(path string) (string, error) {
	corrected, err := resolveUnpkgPath(path)
	if err != nil {
		return ``, err
	}
	if path != corrected {
		// println(`..`, path, `->`, corrected)
		path = corrected
	}
	meta, err := fetchUnpkgMeta(path)
	if err != nil {
		return ``, err
	}

	var template string
	contentType := strings.SplitN(meta.Type, `;`, 2)[0]
	switch contentType {
	case `text/javascript`, `application/javascript`:
		deferred := ``
		if opt.Defer {
			deferred = `defer ` // mind the space.
		}
		template = `<script ` + deferred + `src="$url" integrity="$integrity" crossorigin="anonymous" referrerpolicy="no-referrer"></script>`
	case `text/css`:
		template = `<link rel="stylesheet" href="$url" integrity="$integrity" crossorigin="anonymous" referrerpolicy="no-referrer">`
	case ``:
		return ``, fmt.Errorf(`no content type; Unpkg has changed its schema again?`)
	default:
		return ``, fmt.Errorf(`unknown content type %q`, contentType)
	}
	return expandHTML(template, map[string]string{
		`path`:      meta.Path,
		`integrity`: meta.Integrity,
		`url`:       `https://unpkg.com/` + path,
	})
}

// resolveUnpkgPath lets unpkg redirect us to the full path, which includes the package, path and version.
func resolveUnpkgPath(path string) (string, error) {
	req, err := http.NewRequest(`GET`, `https://unpkg.com/`+path, nil)
	if err != nil {
		return path, err
	}
	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return path, err
	}
	defer rsp.Body.Close()
	defer io.Copy(io.Discard, rsp.Body)
	return strings.TrimPrefix(rsp.Request.URL.Path, `/`), nil
}

func fetchUnpkgMeta(path string) (*fileMeta, error) {
	m := rxResource.FindStringSubmatch(path)
	if m == nil {
		return nil, fmt.Errorf(`could not parse %q into package, file and version`, path)
	}
	// pkg, version, file := m[1], m[2], m[3]
	pkg, filePath := m[1], m[3]

	var meta packageMeta
	url := `https://unpkg.com/` + pkg + `?meta`
	err := getJSON(&meta, url)
	if err != nil {
		return nil, err
	}
	for i := range meta.Files {
		file := &meta.Files[i]
		if file.Path == filePath {
			return file, nil
		}
	}

	return nil, fmt.Errorf(`could not find path %q in %v`, filePath, url)
}

var rxResource = regexp.MustCompile(`^(@?[^@/]+)(@[^/@]+)?(/.*)$`)

type packageMeta struct {
	Package string
	Version string
	Prefix  string
	Files   []fileMeta
}

type fileMeta struct {
	Path      string
	Size      int64
	Type      string
	Integrity string
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

func getJSON(v any, url string) error {
	req, err := http.NewRequest(`GET`, url, nil)
	if err != nil {
		return err
	}
	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = rsp.Body.Close() }()
	switch rsp.StatusCode {
	case 200:
		err = json.NewDecoder(rsp.Body).Decode(v)
		return err
	default:
		return fmt.Errorf(`%v while fetching %v`, rsp.Status, url)
	}
}
