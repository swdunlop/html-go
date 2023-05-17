package main

import (
	"bytes"
	"compress/gzip"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/swdunlop/html-go"
	"github.com/swdunlop/html-go/tag"
)

func main() {
	http.HandleFunc(`/`, handleMain)
	println(`.. visit http://localhost:8080`)
	http.ListenAndServe(`localhost:8080`, nil)
}

// handleMain will provide a full page view, regardless of method.
func handleMain(w http.ResponseWriter, r *http.Request) {
	provideContent(w, r, 200, "panic(`TODO`)", model.htmlContent())
}

var model = Model{
	Items: []Item{
		{ID: 1, Content: `Item 1`},
		{ID: 2, Content: `Item 2`},
		{ID: 3, Content: `Item 3`},
		{ID: 4, Content: `Item 4`},
	},
}

type Model struct {
	Items []Item `json:"items"`
}

func (m *Model) htmlContent() html.Content {
	return tag.New(`ul`).Add(html.Map(m.Items, func(it Item) (view html.Content) {
		return tag.New(`li`).Add(
			tag.New(`button.done`).Text(`Done`),
			html.Text(it.Content),
			tag.New(`button.edit`).Text(`Edit`),
		)
	}))
}

// PostItem will replace an item in the model based on ID.
func (m *Model) PostItem(id int, item *Item) error {
	n := sort.Search(len(m.Items), func(i int) bool {
		return m.Items[i].ID >= id
	})
	item.ID = id // ensure the item has the correct ID.
	if n < len(m.Items) && m.Items[n].ID == id {
		m.Items = append([]Item(nil), m.Items...) // copy the item list.
		m.Items[n] = *item
		return nil
	}
	return m.PutItem(item)
}

// Put will append an item to the model.
func (m *Model) PutItem(item *Item) error {
	id := 0
	if len(m.Items) > 0 {
		id = m.Items[len(m.Items)-1].ID + 1
	}
	if id == 0 {
		return errors.New(`list is full`)
	}
	item.ID = id
	m.Items = append(m.Items, *item)
	return nil
}

type Item struct {
	ID      int    `json:"id"`
	Content string `json:"content"`
}

// provideContent checks to see if the request is a HX-Requst -- if so, it provides just the content and a title.
func provideContent(w http.ResponseWriter, r *http.Request, status int, titleText string, content ...html.Content) {
	buf := make([]byte, 0, 65536)
	h := r.Header
	if h.Get(`HX-Request`) == `true` {
		buf = html.Append(buf, title.Text(titleText), html.Group(content))
	} else {
		buf = html.Append(buf, beforeTitle, title.Text(titleText), beforeBody, html.Group(content), afterBody)
	}
	if strings.Contains(h.Get(`Accept-Encoding`), `gzip`) {
		w.Header().Set(`Content-Encoding`, `gzip`)
		buf = compress(buf)
	}
	h = w.Header()
	h.Set(`Content-Type`, `text/html; charset=utf-8`)
	h.Set(`Cache-Control`, `no-cache, no-store, must-revalidate`)
	h.Set(`Pragma`, `no-cache`)
	h.Set(`Expires`, `0`)
	h.Set(`Content-Length`, strconv.Itoa(len(buf)))
	w.WriteHeader(status)
	_, _ = w.Write(buf)
}

func compress(buf []byte) []byte {
	var tmp bytes.Buffer
	tmp.Grow(len(buf) + 16)
	w := gzipPool.Get().(*gzip.Writer)
	defer gzipPool.Put(w)
	w.Reset(&tmp)
	w.Write(buf)
	w.Close()
	return tmp.Bytes()
}

var gzipPool = sync.Pool{
	New: func() interface{} { w, _ := gzip.NewWriterLevel(nil, gzip.BestSpeed); return w },
}

var (
	title       = tag.New(`title`)
	beforeTitle = html.HTML(`<!DOCTYPE html><html><head>`)
	beforeBody  = html.HTML(`</head><body>`)
	afterBody   = html.HTML(`</body></html>`)
)

// type List struct {
// 	Items   []Item `json:"items"`
// 	Editing int    `json:"editing"`
// }

// func (v List) New() List {
// 	v.Editing = len(v.Items)
// 	v.Items = append(v.Items, Item(``))
// 	return v
// }

// func (v List) Done(ix int) List {
// 	switch {
// 	case ix < 0:
// 	case ix >= len(v.Items):
// 	default:
// 		v.Items = append(v.Items[:ix], v.Items[ix+1:]...)
// 	}
// 	return v
// }

// func (v List) Edit(ix int) List {
// 	switch {
// 	case ix < 0:
// 	case ix >= len(v.Items):
// 	default:
// 		v.Editing = ix
// 	}
// 	return v
// }

// func (v List) Save(text string) List {
// 	v.Items = append([]Item(nil), v.Items...) // copy the item list.
// 	v.Items[v.Editing] = Item(text)
// 	v.Editing = len(v.Items)
// 	return v
// }

// func (v List) AppendHTML(buf []byte) []byte {
// 	ix := 0
// 	return tag.New(`ul`).Add(html.Map(v.Items, func(it Item) (view html.Content) {
// 		if ix == v.Editing {
// 			view = it.editor()
// 		} else {
// 			view = it.viewer()
// 		}
// 		ix++
// 		return
// 	})).AppendHTML(buf)
// }

// // Item represents a single todo item.
// type Item string

// func (it Item) editor() html.Content {
// 	return tag.New(`li`).Add(
// 		doneButton,
// 		tag.New(`span[contenteditable=true]`).Text(it),
// 		saveButton,
// 	)
// }

// func (it Item) viewer() html.Content {
// 	return tag.New(`li`).Add(
// 		doneButton,
// 		html.Text(it),
// 		editButton,
// 	)
// }

// // unsortedList creates unsorted list tags.
// var unsortedList = tag.New(`ul`)

// // listItem creates list item tags.
// var listItem = tag.New(`li`)

// // editableText wraps text into an span that is content editable.
// var editableText = tag.New(`span[contenteditable=true]`)

// var (
// 	// newButton is a static "New" button with id class "new".
// 	newButton = html.Static(tag.New(`button.new`).Text(`New`))

// 	// doneButton is a static "Done" button with class "done".
// 	doneButton = html.Static(tag.New(`button.done`).Text(`Done`))

// 	// editButton is a static "Edit" button with class "edit".
// 	editButton = html.Static(tag.New(`button.edit`).Text(`Edit`))

// 	// saveButton is a static "Save" button with class "save".
// 	saveButton = html.Static(tag.New(`button.save`).Text(`Save`))
// )
