package datastar

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/swdunlop/html-go"
)

func TestDecode(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		contentType string
		body        string
		queryParam  string
		expected    map[string]interface{}
		expectError bool
	}{
		{
			name:       "GET with datastar query param",
			method:     "GET",
			queryParam: `{"user":"test","count":42}`,
			expected:   map[string]interface{}{"user": "test", "count": float64(42)},
		},
		{
			name:        "POST with JSON body",
			method:      "POST",
			contentType: "application/json",
			body:        `{"user":"test","count":42}`,
			expected:    map[string]interface{}{"user": "test", "count": float64(42)},
		},
		{
			name:        "PUT with JSON body",
			method:      "PUT",
			contentType: "application/json",
			body:        `{"action":"update"}`,
			expected:    map[string]interface{}{"action": "update"},
		},
		{
			name:        "POST with unsupported content type",
			method:      "POST",
			contentType: "text/plain",
			body:        "plain text",
			expectError: true,
		},
		{
			name:        "POST with no content type",
			method:      "POST",
			body:        `{"test":"value"}`,
			expectError: true,
		},
		{
			name:        "GET with invalid JSON in query param",
			method:      "GET",
			queryParam:  `{invalid json}`,
			expectError: true,
		},
		{
			name:        "POST with invalid JSON body",
			method:      "POST",
			contentType: "application/json",
			body:        `{invalid json}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.method == "GET" {
				u := &url.URL{Path: "/test"}
				if tt.queryParam != "" {
					u.RawQuery = "datastar=" + url.QueryEscape(tt.queryParam)
				}
				req = httptest.NewRequest(tt.method, u.String(), nil)
			} else {
				req = httptest.NewRequest(tt.method, "/test", strings.NewReader(tt.body))
				if tt.contentType != "" {
					req.Header.Set("Content-Type", tt.contentType)
				}
			}

			var result map[string]interface{}
			err := Decode(&result, req)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d fields, got %d", len(tt.expected), len(result))
			}

			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("expected %s=%v, got %v", k, v, result[k])
				}
			}
		})
	}
}

func TestEncode(t *testing.T) {
	tests := []struct {
		name        string
		accept      string
		data        interface{}
		expectError bool
	}{
		{
			name:   "accepts application/json",
			accept: "application/json",
			data:   map[string]string{"test": "value"},
		},
		{
			name:   "accepts application/*",
			accept: "application/*",
			data:   map[string]string{"test": "value"},
		},
		{
			name:   "accepts */*",
			accept: "*/*",
			data:   map[string]string{"test": "value"},
		},
		{
			name:   "no accept header",
			accept: "",
			data:   map[string]string{"test": "value"},
		},
		{
			name:        "rejects text/plain",
			accept:      "text/plain",
			data:        map[string]string{"test": "value"},
			expectError: true,
		},
		{
			name:        "rejects text/html",
			accept:      "text/html",
			data:        map[string]string{"test": "value"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.accept != "" {
				req.Header.Set("Accept", tt.accept)
			}

			w := httptest.NewRecorder()
			err := Encode(w, req, tt.data)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if w.Code != 200 {
				t.Errorf("expected status 200, got %d", w.Code)
			}

			if ct := w.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", ct)
			}

			var result map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
				t.Errorf("failed to unmarshal response: %v", err)
			}
		})
	}
}

func TestRequestStream(t *testing.T) {
	tests := []struct {
		name        string
		accept      string
		expectError bool
	}{
		{
			name:   "accepts text/event-stream",
			accept: "text/event-stream",
		},
		{
			name:   "accepts text/*",
			accept: "text/*",
		},
		{
			name:   "accepts */*",
			accept: "*/*",
		},
		{
			name:   "no accept header",
			accept: "",
		},
		{
			name:        "rejects application/json",
			accept:      "application/json",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.accept != "" {
				req.Header.Set("Accept", tt.accept)
			}

			w := httptest.NewRecorder()
			stream, err := RequestStream(w, req)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if stream == nil {
				t.Errorf("expected stream, got nil")
			}

			if ct := w.Header().Get("Content-Type"); ct != "text/event-stream" {
				t.Errorf("expected Content-Type text/event-stream, got %s", ct)
			}

			if cc := w.Header().Get("Cache-Control"); cc != "no-cache" {
				t.Errorf("expected Cache-Control no-cache, got %s", cc)
			}

			if conn := w.Header().Get("Connection"); conn != "keep-alive" {
				t.Errorf("expected Connection keep-alive, got %s", conn)
			}
		})
	}
}

func TestElements(t *testing.T) {
	tests := []struct {
		name     string
		content  html.Content
		options  []ElementsOption
		expected string
	}{
		{
			name:     "basic elements",
			content:  html.HTML("<div>Hello</div>"),
			expected: "event: datastar-patch-elements\ndata: elements <div>Hello</div>\n",
		},
		{
			name:     "elements with mode",
			content:  html.HTML("<div>Hello</div>"),
			options:  []ElementsOption{Mode("morph")},
			expected: "event: datastar-patch-elements\ndata: mode morph\ndata: elements <div>Hello</div>\n",
		},
		{
			name:     "elements with selector",
			content:  html.HTML("<div>Hello</div>"),
			options:  []ElementsOption{Selector("#content")},
			expected: "event: datastar-patch-elements\ndata: selector #content\ndata: elements <div>Hello</div>\n",
		},
		{
			name:     "elements with mode and selector",
			content:  html.HTML("<div>Hello</div>"),
			options:  []ElementsOption{Mode("morph"), Selector("#content")},
			expected: "event: datastar-patch-elements\ndata: mode morph\ndata: selector #content\ndata: elements <div>Hello</div>\n",
		},
		{
			name:     "elements with newlines",
			content:  html.HTML("<div>\nHello\nWorld\n</div>"),
			expected: "event: datastar-patch-elements\ndata: elements <div>&#10;Hello&#10;World&#10;</div>\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := Elements(tt.content, tt.options...)
			var buf []byte
			buf = event.appendEvent(buf)
			result := string(buf)

			if result != tt.expected {
				t.Errorf("expected:\n%q\ngot:\n%q", tt.expected, result)
			}
		})
	}
}

func TestSignal(t *testing.T) {
	tests := []struct {
		name     string
		data     interface{}
		expected string
	}{
		{
			name:     "simple signal",
			data:     map[string]interface{}{"user": "test", "count": 42},
			expected: "event: datastar-patch-signals\ndata: signals {\"count\":42,\"user\":\"test\"}\n\n",
		},
		{
			name:     "string signal",
			data:     "hello",
			expected: "event: datastar-patch-signals\ndata: signals \"hello\"\n\n",
		},
		{
			name:     "number signal",
			data:     42,
			expected: "event: datastar-patch-signals\ndata: signals 42\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := Signal(tt.data)
			var buf []byte
			buf = event.appendEvent(buf)
			result := string(buf)

			if result != tt.expected {
				t.Errorf("expected:\n%q\ngot:\n%q", tt.expected, result)
			}
		})
	}
}

func TestSignalIfMissing(t *testing.T) {
	tests := []struct {
		name     string
		data     interface{}
		expected string
	}{
		{
			name:     "simple signal if missing",
			data:     map[string]interface{}{"user": "test", "count": 42},
			expected: "event: datastar-patch-signals\ndata: onlyIfMissing true\ndata: signals {\"count\":42,\"user\":\"test\"}\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := SignalIfMissing(tt.data)
			var buf []byte
			buf = event.appendEvent(buf)
			result := string(buf)

			if result != tt.expected {
				t.Errorf("expected:\n%q\ngot:\n%q", tt.expected, result)
			}
		})
	}
}

func TestBatch(t *testing.T) {
	event1 := Signal(map[string]string{"user": "test"})
	event2 := Elements(html.HTML("<div>Hello</div>"))

	batch := Batch(event1, event2)

	var buf []byte
	buf = batch.appendEvent(buf)
	result := string(buf)

	expected1 := "event: datastar-patch-signals\ndata: signals {\"user\":\"test\"}\n\n"
	expected2 := "event: datastar-patch-elements\ndata: elements <div>Hello</div>\n"
	expected := expected1 + expected2

	if result != expected {
		t.Errorf("expected:\n%q\ngot:\n%q", expected, result)
	}
}

func TestStreamEmit(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept", "text/event-stream")

	w := httptest.NewRecorder()
	stream, err := RequestStream(w, req)
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	event1 := Signal(map[string]string{"user": "test"})
	event2 := Elements(html.HTML("<div>Hello</div>"))

	err = stream.Emit(event1, event2)
	if err != nil {
		t.Errorf("failed to emit events: %v", err)
	}

	body := w.Body.String()
	if !strings.Contains(body, "event: datastar-patch-signals") {
		t.Errorf("expected signal event in output")
	}
	if !strings.Contains(body, "event: datastar-patch-elements") {
		t.Errorf("expected elements event in output")
	}
}

func TestModePanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic for mode with newline")
		}
	}()
	Mode("invalid\nmode")
}

func TestSelectorPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic for selector with newline")
		}
	}()
	Selector("invalid\nselector")
}

func TestAcceptsContentTypes(t *testing.T) {
	tests := []struct {
		name         string
		acceptHeader string
		contentTypes []string
		expected     bool
	}{
		{
			name:         "no accept header",
			acceptHeader: "",
			contentTypes: []string{"application/json"},
			expected:     true,
		},
		{
			name:         "exact match",
			acceptHeader: "application/json",
			contentTypes: []string{"application/json"},
			expected:     true,
		},
		{
			name:         "wildcard match",
			acceptHeader: "application/*",
			contentTypes: []string{"application/json"},
			expected:     true,
		},
		{
			name:         "universal wildcard",
			acceptHeader: "*/*",
			contentTypes: []string{"application/json"},
			expected:     true,
		},
		{
			name:         "multiple accepts with match",
			acceptHeader: "text/html,application/json,*/*;q=0.8",
			contentTypes: []string{"application/json"},
			expected:     true,
		},
		{
			name:         "no match",
			acceptHeader: "text/html",
			contentTypes: []string{"application/json"},
			expected:     false,
		},
		{
			name:         "quality values ignored",
			acceptHeader: "application/json;q=0.8,text/html;q=0.9",
			contentTypes: []string{"application/json"},
			expected:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.acceptHeader != "" {
				req.Header.Set("Accept", tt.acceptHeader)
			}

			result := acceptsContentTypes(req, tt.contentTypes...)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestHTTPError(t *testing.T) {
	originalErr := fmt.Errorf("original error")
	httpErr := httpError{status: 404, err: originalErr}

	if httpErr.HTTPStatus() != 404 {
		t.Errorf("expected status 404, got %d", httpErr.HTTPStatus())
	}

	if httpErr.Error() != "original error" {
		t.Errorf("expected error message 'original error', got %s", httpErr.Error())
	}

	if httpErr.Unwrap() != originalErr {
		t.Errorf("expected unwrapped error to be original error")
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"test": "value"}

	writeJSON(w, 201, data)

	if w.Code != 201 {
		t.Errorf("expected status 201, got %d", w.Code)
	}

	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}

	var result map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Errorf("failed to unmarshal response: %v", err)
	}

	if result["test"] != "value" {
		t.Errorf("expected test=value, got %s", result["test"])
	}
}

func TestWriteText(t *testing.T) {
	w := httptest.NewRecorder()
	text := "hello world"

	writeText(w, 201, text)

	if w.Code != 201 {
		t.Errorf("expected status 201, got %d", w.Code)
	}

	if ct := w.Header().Get("Content-Type"); ct != "text/plain" {
		t.Errorf("expected Content-Type text/plain, got %s", ct)
	}

	if body := w.Body.String(); body != text {
		t.Errorf("expected body %s, got %s", text, body)
	}
}
