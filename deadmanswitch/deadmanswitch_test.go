package deadmanswitch

import (
	"strings"
	"testing"
)

// TestScriptLifecycle pins the parts of the generated script that keep the
// switch from leaking sockets: the EventSource must be closed when the page is
// hidden and reopened when it is shown again, and a page restored from the
// back/forward cache must re-enter through the reconnect state so OnReconnect
// hooks fire.
func TestScriptLifecycle(t *testing.T) {
	html := string(New(ReloadOnReconnect()).AppendHTML(nil))
	for _, want := range []string{
		`new EventSource("/dead-man-switch")`,
		`addEventListener('pagehide'`,
		`dms.sse.close()`,
		`addEventListener('pageshow'`,
		`if (evt.persisted && dms.connected == true) dms.connected = false;`,
		`window.dms.on.reconnect.push(function(){window.location.reload()});`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("generated script is missing %q", want)
		}
	}
	if open, close := strings.Count(html, "{"), strings.Count(html, "}"); open != close {
		t.Errorf("unbalanced braces in generated script: %d open, %d close", open, close)
	}
}
