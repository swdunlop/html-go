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
	content := make([]html.Element, len(v.Items))
	for i, it := range v.Items {
		if i == v.Editing {
			content[i] = it.editor()
		} else {
			content[i] = it.viewer()
		}
	}
	return unsortedList(tag.Content(content...), newButton).AppendHTML(buf)
}

// Item represents a single todo item.
type Item string

func (it Item) editor() html.Tag {
	return listItem(
		editableText(string(it)),
		saveButton,
	)
}

func (it Item) viewer() html.Tag {
	return listItem(
		doneButton,
		tag.Text(string(it)),
		editButton,
	)
}

// unsortedList creates unsorted list tags.
var unsortedList = tag.Factory(`ul`)

// listItem creates list item tags.
var listItem = tag.Factory(`li`)

// editableText wraps text into an span that is content editable.
func editableText(text string, options ...tag.Option) tag.Option {
	return tag.Content(tag.New(`span`, tag.Text(text), contentEditable, tag.Apply(options...)))
}

// contentEditable sets the content editable attribute to true on a tag.
var contentEditable = tag.Attr(`contenteditable`, `true`)

// newButton is a static "New" button with id class "new".
var newButton = tag.Static(button(`new`, `New`))

// doneButton is a static "Done" button with class "done".
var doneButton = tag.Static(button(`done`, `Done`))

// editButton is a static "Edit" button with class "edit".
var editButton = tag.Static(button(`edit`, `Edit`))

// saveButton is a static "Save" button with class "save".
var saveButton = tag.Static(button(`save`, `Save`))

// button creates a button tag with a class and label.
func button(class, label string, options ...tag.Option) html.Tag {
	return tag.New(`button`, tag.Attr(`class`, class), tag.Text(label))
}
