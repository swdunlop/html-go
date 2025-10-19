// Package datastar provides some support for Datastar (there is more in the official Go SDK).  Note that datastar
// clients accept more than just JSON and SSE -- text/javascript and text/html is also supported by Datastar, so not
// all cases require the use of Handle or HandleStream.
//
// The major difference between this package and the official SDK is the Stream and Event interface.  The SDK currently
// uses fmt.Sprintf in some worrying places that is inefficient at best and may be susceptible to XSS.  It supports a
// lot more of Datastar than this package, however.
package datastar

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/swdunlop/html-go"
)

// Decode decodes the request body if the method is not GET, or the datastar query parameter.  This will return an
// error if the body could not be decoded.  This function does not currently support form based input.
func Decode(data any, r *http.Request) error {
	if r.Method == `GET` {
		return json.NewDecoder(
			strings.NewReader(r.URL.Query().Get(`datastar`)),
		).Decode(data)
	} else if contentType := r.Header.Get(`Content-Type`); contentType == `application/json` {
		return json.NewDecoder(r.Body).Decode(data)
	} else {
		return fmt.Errorf(`unsupported content type %q for method %q`, contentType, r.Method)
	}
}

// Encode encodes the response body, checking that the client accepts either application/json, application/* or
// */*.  If the body cannot be encoded, or the client does not accept the encoding, an error is returned.
func Encode(w http.ResponseWriter, r *http.Request, data any) error {
	if !acceptsJSON(r) {
		return fmt.Errorf(`client does not accept JSON`)
	}
	writeJSON(w, 200, data)
	return nil
}

// RequestStream examines a http.Request and http.ResponseWriter, and if possible, returns a Stream that supports
// emitting Datastar events.  This will return an error if the request does not accept SSE, or if the response writer
// cannot be flushed (a requirement for SSE in Go, since requests may otherwise buffer in ways that interfere with
// streaming events).
//
// If this returns a stream, content cannot be written to the underlying writer using Write.
func RequestStream(w http.ResponseWriter, r *http.Request) (Stream, error) {
	if !acceptsSSE(r) {
		return nil, fmt.Errorf(`client does not accept SSE`)
	}
	wf, ok := w.(writeFlusher)
	if !ok {
		return nil, fmt.Errorf(`response writer cannot be flushed`)
	}
	return stream{make([]byte, 0, 16384), wf}, startSSE(wf)
}

func startSSE(wf writeFlusher) error {
	h := wf.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	wf.WriteHeader(200)
	wf.Flush()
	return nil
}

type writeFlusher interface {
	http.Flusher
	http.ResponseWriter
}

type httpError struct {
	status int
	err    error
}

func (err httpError) Unwrap() error   { return err.err }
func (err httpError) Error() string   { return err.err.Error() }
func (err httpError) HTTPStatus() int { return err.status }

type stream struct {
	buf []byte
	out writeFlusher
}

func (sm stream) Emit(events ...Event) error {
	buf := sm.buf
	defer func() {
		sm.buf = buf[0:0:cap(buf)]
	}()
	for _, event := range events {
		buf = event.appendEvent(buf)
	}
	_, err := sm.out.Write(buf)
	if err == nil {
		sm.out.Flush()
	}
	return err
}

// Stream describes a stream of Server Sent Events that will be sent to a Datastar client.
type Stream interface {
	// Emit sends a batch of Datastar events to the client.  This will return an error if the emit times out.
	Emit(events ...Event) error
}

// Batch takes a set of events and makes a static []byte that is (marginally) faster to send.
func Batch(events ...Event) Event {
	buf := make([]byte, 0, 1024)
	for _, event := range events {
		buf = event.appendEvent(buf)
	}
	return batch(buf)
}

type batch []byte

func (evt batch) appendEvent(buf []byte) []byte { return append(buf, evt...) }

// Elements produces a Datastar event that tells Datastar to patch elements presented by the client.
//
// See https://data-star.dev/reference/sse_events#datastar-patch-elements
func Elements(content html.Content, options ...ElementsOption) Event {
	evt := elements{content: content}
	for _, option := range options {
		option(&evt)
	}
	return &evt
}

// Mode affects how elements are patched by Datastar.  The last Mode specified as an option "wins."
//
// This will panic if the mode contains a newline.
func Mode(mode string) ElementsOption {
	if strings.Contains(mode, "\n") {
		panic(errors.New(`Modes cannot contain newlines`))
	}
	return func(p *elements) { p.mode = mode }
}

// Selector affects which elements are patched by Datastar.  The last Selector specified as an option "wins."
//
// This will panic if the selector contains a newline.
func Selector(selector string) ElementsOption {
	if strings.Contains(selector, "\n") {
		panic(errors.New(`Selectors cannot contain newlines`))
	}
	return func(p *elements) { p.selector = selector }
}

// ElementsOption affects how elements are patched by the Datastar client.
type ElementsOption func(*elements)

type elements struct {
	content  html.Content
	mode     string
	selector string
}

func (p *elements) appendEvent(buf []byte) []byte {
	const (
		eventPrefix    = "event: datastar-patch-elements"
		modePrefix     = "\ndata: mode "
		selectorPrefix = "\ndata: selector "
		elementsPrefix = "\ndata: elements "
	)

	// First generate the HTML content
	contentBytes := p.content.AppendHTML(nil)

	sz := len(eventPrefix) + len(elementsPrefix) + len(contentBytes) + 1
	if p.mode != `` {
		sz += len(modePrefix) + len(p.mode)
	}
	if p.selector != `` {
		sz += len(selectorPrefix) + len(p.selector)
	}

	buf = slices.Grow(buf, sz)
	buf = append(buf, eventPrefix...)
	if p.mode != `` {
		buf = append(buf, modePrefix...)
		buf = append(buf, p.mode...)
	}
	if p.selector != `` {
		buf = append(buf, selectorPrefix...)
		buf = append(buf, p.selector...)
	}

	buf = append(buf, elementsPrefix...)
	for len(contentBytes) > 0 {
		ofs := bytes.IndexByte(contentBytes, '\n')
		if ofs < 0 {
			buf, contentBytes = append(buf, contentBytes...), nil
		} else {
			buf, contentBytes = append(buf, contentBytes[:ofs]...), contentBytes[ofs+1:]
			buf = append(buf, `&#10;`...)
		}
	}
	buf = append(buf, '\n', '\n')
	return buf
}

// Signal produces a Datastar event that tells Datastar to patch the client state.
//
// See https://data-star.dev/reference/sse_events#datastar-patch-signals
func Signal(v any) Event { return signal{false, v} }

// SignalIfMissing produces a Datastar event that tells Datastar to patch the client state where it is missing.
//
// See https://data-star.dev/reference/sse_events#datastar-patch-signals
func SignalIfMissing(v any) Event { return signal{true, v} }

type signal struct {
	onlyIfMissing bool
	data          any
}

func (evt signal) appendEvent(buf []byte) []byte {
	header := "event: datastar-patch-signals\ndata: signals "
	if evt.onlyIfMissing {
		header = "event: datastar-patch-signals\ndata: onlyIfMissing true\ndata: signals "
	}
	js, err := json.Marshal(evt.data)
	if err != nil {
		panic(err)
	}
	sz := len(header) + len(js) + 2
	buf = slices.Grow(buf, sz)
	buf = append(buf, header...)
	buf = append(buf, js...)
	buf = append(buf, '\n', '\n')
	return buf
}

// An Event describes an event that can be sent to a Datastar client via SSE.  Various utility functions in this
// package produce events.
type Event interface {
	// appendEvent appends the event to a buffer of server sent events for output.
	appendEvent(buf []byte) []byte

	// see https://data-star.dev/reference/sse_events for the events Datastar supports.
}

// appendEventType appends the event type to a buffer of server sent events for output.  This does not check for
// newlines, therefore eventType must be well controlled.
func appendEventType(buf []byte, eventType string) []byte {
	buf = append(buf, `event: `...)
	buf = append(buf, eventType...)
	buf = append(buf, '\n')
	return buf
}

// appendEventElement appends HTML elements to an event; unlike many other of the appendEvent utilities, this WILL
// check for newlines and encode them using HTML entities.
func appendEventElement(buf []byte, element []byte) []byte {
	buf = append(buf, `data: elements `...)
	for len(element) > 0 {
		ofs := bytes.IndexByte(element, '\n')
		if ofs < 0 {
			buf, element = append(buf, element...), nil
		} else {
			buf, element = append(buf, element[:ofs]...), element[ofs+1:]
			// according to https://data-star.dev/reference/sse_events#datastar-patch-elements
			// we could also use "\ndata: elements "
			buf = append(buf, `&#10;`...)
		}
	}
	buf = append(buf, '\n')
	// TODO: check the upstream Go implementation of the SDK, and file a PR if they aren't checking for newlines in
	// the patch -- this could lead to some pretty tricky event smuggling attacks in Datastar applications.
	return buf
}

// appendEventMode appends mode data to an event, this does not check the selector for newlines
func appendEventSelector(buf []byte, selector string) []byte {
	buf = append(buf, `data: selector `...)
	buf = append(buf, selector...)
	buf = append(buf, '\n')
	return buf
}

// appendEventMode appends mode data to an event, this does not check the mode for newlines.
func appendEventMode(buf []byte, mode string) []byte {
	buf = append(buf, `data: mode `...)
	buf = append(buf, mode...)
	buf = append(buf, '\n')
	return buf
}

// appendEventString appends data to an event to a buffer of server sent events for output.  This does not check dataType
// or data for newlines
func appendEventString(buf []byte, dataType string, data string) []byte {
	buf = append(buf, `data: `...)
	buf = append(buf, dataType...)
	buf = append(buf, ' ')
	buf = append(buf, data...)
	buf = append(buf, '\n')
	return buf
}

// appendEventBytes appends data to an event to a buffer of server sent events for output.  This does not check dataType
// or data for newlines
func appendEventBytes(buf []byte, dataType string, data []byte) []byte {
	buf = append(buf, `data: `...)
	buf = append(buf, dataType...)
	buf = append(buf, ' ')
	buf = append(buf, data...)
	buf = append(buf, '\n')
	return buf
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	status := 500
	if impl, ok := err.(interface{ HTTPStatus() int }); ok {
		status = impl.HTTPStatus()
	}
	if acceptsJSON(r) {
		writeJSON(w, status, struct {
			Status int   `json:"status"`
			Err    error `json:"error"`
		}{status, err})
	} else {
		writeText(w, status, err.Error())
	}
}

func acceptsJSON(r *http.Request) bool {
	return acceptsContentTypes(r, `application/json`, `application/*`, `*/*`)
}

func acceptsSSE(r *http.Request) bool {
	return acceptsContentTypes(r, `text/event-stream`, `text/*`, `*/*`)
}

func acceptsContentTypes(r *http.Request, contentTypes ...string) bool {
	headers := r.Header[`Accept`]
	if len(headers) == 0 {
		return true // dumb client, probably netcat, probably accepts anything.
	}

	for _, header := range headers {
		for _, accept := range strings.Split(header, `,`) {
			accept = strings.SplitN(accept, `;`, 2)[0]
			accept = strings.TrimSpace(accept)
			if slices.Contains(contentTypes, accept) {
				return true
			}
			// Check for wildcard matches
			if accept == "*/*" {
				return true
			}
			if strings.HasSuffix(accept, "/*") {
				prefix := accept[:len(accept)-1]
				for _, ct := range contentTypes {
					if strings.HasPrefix(ct, prefix) {
						return true
					}
				}
			}
		}
	}

	return false
}

func writeJSON(w http.ResponseWriter, httpStatus int, data any) {
	msg, err := json.Marshal(data)
	if err != nil {
		panic(err) // should not happen.
	}
	h := w.Header()
	h.Set(`Content-Type`, `application/json`)
	h.Set(`Content-Length`, strconv.Itoa(len(msg)))
	w.WriteHeader(httpStatus)
	_, _ = w.Write(msg)
}

func writeText(w http.ResponseWriter, httpStatus int, text string) {
	h := w.Header()
	h.Set(`Content-Type`, `text/plain`)
	h.Set(`Content-Length`, strconv.Itoa(len(text)))
	w.WriteHeader(httpStatus)
	_, _ = w.Write([]byte(text))
}
