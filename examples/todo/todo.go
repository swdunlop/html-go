package main

import (
	"github.com/swdunlop/html-go"
	"github.com/swdunlop/html-go/tag"
)

type List struct {
	Items   []Item `json:"items"`
	Editing int    `json:"editing"`
}

func (v List) New() List {
	v.Editing = len(v.Items)
	v.Items = append(v.Items, Item(``))
	return v
}

func (v List) Done(ix int) List {
	switch {
	case ix < 0:
	case ix >= len(v.Items):
	default:
		v.Items = append(v.Items[:ix], v.Items[ix+1:]...)
	}
	return v
}

func (v List) Edit(ix int) List {
	switch {
	case ix < 0:
	case ix >= len(v.Items):
	default:
		v.Editing = ix
	}
	return v
}

func (v List) Save(text string) List {
	v.Items = append([]Item(nil), v.Items...) // copy the item list.
	v.Items[v.Editing] = Item(text)
	v.Editing = len(v.Items)
	return v
}

func (v List) AppendHTML(buf []byte) []byte {
	ix := 0
	return tag.New(`ul`).Add(html.Map(v.Items, func(it Item) (view html.Content) {
		if ix == v.Editing {
			view = it.editor()
		} else {
			view = it.viewer()
		}
		ix++
		return
	})).AppendHTML(buf)
}

// Item represents a single todo item.
type Item string

func (it Item) editor() html.Content {
	return tag.New(`li`).Add(
		doneButton,
		tag.New(`span[contenteditable=true]`).Text(it),
		saveButton,
	)
}

func (it Item) viewer() html.Content {
	return tag.New(`li`).Add(
		doneButton,
		html.Text(it),
		editButton,
	)
}

// unsortedList creates unsorted list tags.
var unsortedList = tag.New(`ul`)

// listItem creates list item tags.
var listItem = tag.New(`li`)

// editableText wraps text into an span that is content editable.
var editableText = tag.New(`span[contenteditable=true]`)

var (
	// newButton is a static "New" button with id class "new".
	newButton = html.Static(tag.New(`button.new`).Text(`New`))

	// doneButton is a static "Done" button with class "done".
	doneButton = html.Static(tag.New(`button.done`).Text(`Done`))

	// editButton is a static "Edit" button with class "edit".
	editButton = html.Static(tag.New(`button.edit`).Text(`Edit`))

	// saveButton is a static "Save" button with class "save".
	saveButton = html.Static(tag.New(`button.save`).Text(`Save`))
)
