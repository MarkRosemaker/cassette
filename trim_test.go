package cassette_test

import (
	"net/http"
	"testing"

	"github.com/MarkRosemaker/cassette"
)

func TestTrimResponseHeaders(t *testing.T) {
	t.Parallel()

	ias := cassette.Interactions{{Response: cassette.Response{Headers: http.Header{
		"Content-Type": {"application/json"},
		"Date":         {"Sat, 10 Oct 2026 12:00:00 GMT"},
	}}}}

	ias.TrimResponseHeaders()

	if got, want := len(ias[0].Response.Headers), 1; got != want {
		t.Fatalf("got %d headers, want %d: %v", got, want, ias[0].Response.Headers)
	}

	if got := ias[0].Response.Headers.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want it kept", got)
	}
}
