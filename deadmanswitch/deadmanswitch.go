// Package deadmanswitch provides a component that can be used to run JavaScript expressions when a Server Sent Events
// (SSE) connection to a service is lost.  This is useful for reloading HTML views when the server restarts.
package deadmanswitch

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/swdunlop/html-go"
)

// New returns a new Dead Man's Switch which can handle inbound Server Sent Events (SSE) connections and provides
// HTML content containing a script tag that detects when the SSE connection is lost and when it returns.  This is
// useful for reloading content when the server restarts.
func New(options ...Option) Interface {
	cfg := &config{path: "/dead-man-switch"}
	for _, option := range options {
		option(cfg)
	}
	cfg.assembleHTML()
	return cfg
}

// ReloadOnReconnect uses window.location.reload when the switch reconnects.  This should refresh the window
// without dependencies.  For libraries like HTMX or Unpoly, there is usually a better way to do this that will not
// cause a full refresh.
func ReloadOnReconnect() Option {
	return OnReconnect(`window.location.reload()`)
}

// OnConnect appends one or more more JavaScript scripts that will be run when the client has connected.
// Each script is wrapped in a JavaScript function, isolating it from the other scripts.
func OnConnect(expr ...string) Option {
	return on(`connect`, expr...)
}

// OnDisconnect appends one or more JavaScript scripts that will be run when the client has disconnected.
// Each script is wrapped in a JavaScript function, isolating it from the other scripts.
func OnDisconnect(expr ...string) Option {
	return on(`disconnect`, expr...)
}

// OnReconnect appends one or more JavaScript scripts that will be run when the client has disconnected.
// Each script is wrapped in a JavaScript function, isolating it from the other scripts.
func OnReconnect(expr ...string) Option {
	return on(`reconnect`, expr...)
}

func on(hook string, expr ...string) Option {
	if len(expr) == 0 {
		return withSwitch()
	}
	var buf strings.Builder
	buf.WriteString(`window.dms.on.`)
	buf.WriteString(hook)
	buf.WriteString(`.push(function(){`)
	buf.WriteString(expr[0])
	for _, expr := range expr[1:] {
		buf.WriteString(`},`)
		buf.WriteString(expr)
	}
	buf.WriteString(`});`)
	return withSwitch(buf.String())
}

func withSwitch(expr ...string) Option {
	return func(cfg *config) {
		cfg.exprs = append(cfg.exprs, expr...)
	}
}

// OnDisconnect appends one or more JavaScript expressions that will be run when the client has disconnected.

// Path specifies the path to the Dead Man's Switch handler.  By default, this is "/dead-man-switch".
func Path(path string) Option { return func(cfg *config) { cfg.path = path } }

// An Option affects the configuration of a new Dead Man's Switch.
type Option func(*config)

// Interface describes the methods provided by a configured Dead Man's Switch.  You should mount this in your HTTP
// router where Path was configured, by default this is "/dead-man-switch".
type Interface interface {
	html.Content
	http.Handler

	// Path returns the path where the handler should be mounted.
	Path() string
}

type config struct {
	path  string
	exprs []string
	html  []byte
}

// Path implements Interface by returning the expected path for SSE connections.
func (cfg *config) Path() string { return cfg.path }

// AppendHTML implements html.Content by appendin
func (cfg *config) AppendHTML(p []byte) []byte { return append(p, cfg.html...) }

// ServeHTTP implements http.Handler by accepting inbound SSE connections and holding them until the provided context
// is cancelled or the connection is lost.
func (cfg *config) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Set(`Content-Type`, `text/event-stream`)
	h.Set(`Cache-Control`, `no-cache`)
	h.Set(`Connection`, `keep-alive`)
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, `streaming unsupported`, http.StatusInternalServerError)
		return
	}
	w.Write([]byte("event: connected\r\n\r\n"))
	flusher.Flush()
	<-r.Context().Done()
}

func (cfg *config) assembleHTML() {
	var buf bytes.Buffer
	buf.WriteString(beforePath)
	p, err := json.Marshal(cfg.path)
	if err != nil {
		panic(err)
	}
	buf.Write(p)
	buf.WriteString(afterPath)
	for _, expr := range cfg.exprs {
		buf.WriteByte('\t')
		buf.WriteString(expr)
		buf.WriteByte('\n')
	}
	buf.WriteString(afterExprs)
	cfg.html = buf.Bytes()
}

const (
	beforePath = `<script>(function(){
	if (window.dms != undefined) return;
	const sse = new EventSource(`

	afterPath = `);
	const dms = {on: {connect: [], disconnect: [], reconnect: []}, connected: null, sse: sse};
	window.dms = dms;
	const run = function(hook) { dms.on[hook].map(function(f) { f(); }); };
	sse.addEventListener('open', function(){
		if (dms.connected == true) return;
		try {
			if (dms.connected == null) { run('connect') } else { run('reconnect') };
		} finally {
			dms.connected = true;
		};
	});
	sse.addEventListener('error', function(){
		if (dms.connected != true) return;
		try { run('disconnect'); } finally { dms.connected = false; };
	});
`
	afterExprs = "})()</script>\n"
)
