package cassette_test

import (
	"bytes"
	"encoding/json/jsontext"
	"io"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/MarkRosemaker/cassette"
)

// sameJSON reports whether got is body read back: the same JSON, however indented, or body written as a JSON string
// where it is no JSON.
func sameJSON(got cassette.Body, body string) bool {
	want := jsontext.Value(body)
	if !want.IsValid() {
		want, _ = jsontext.AppendQuote(nil, body)
	}

	g := jsontext.Value(bytes.Clone(got))

	return g.Compact() == nil && want.Compact() == nil && bytes.Equal(g, want)
}

// interaction is a call with the request and response bodies given, nil for none.
func interaction(t *testing.T, reqBody, rspBody []byte) *cassette.Interaction {
	t.Helper()

	var body io.Reader = http.NoBody
	if reqBody != nil {
		body = bytes.NewReader(reqBody)
	}

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "https://example.com/things", body)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/json")

	rec, err := cassette.NewRequest(req)
	if err != nil {
		t.Fatal(err)
	}

	return &cassette.Interaction{Request: rec, Response: cassette.Response{StatusCode: http.StatusOK, Body: rspBody}}
}

// Every body is written and read back as it was, an empty JSON value as much as any other, so AddInteraction knows a
// call it holds already and adds it once.
func TestAddInteraction_Bodies(t *testing.T) {
	t.Parallel()

	for _, body := range []string{`null`, `{}`, `[]`, `""`, `0`, `false`, `{"a":1}`, `not json`} {
		t.Run(body, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "interactions.json")
			ia := interaction(t, []byte(body), []byte(body))

			for range 2 {
				if err := cassette.AddInteraction(path, ia); err != nil {
					t.Fatal(err)
				}
			}

			ias, err := cassette.InteractionsReadFile(path)
			if err != nil {
				t.Fatal(err)
			}

			if len(ias) != 1 {
				t.Fatalf("the same call added twice is held %d times, want once", len(ias))
			}

			for name, got := range map[string]cassette.Body{"request": ias[0].Request.Body, "response": ias[0].Response.Body} {
				if !sameJSON(got, body) {
					t.Errorf("%s body read back as %q, want %q", name, got, body)
				}
			}
		})
	}
}

// A call with no body at all is written without one, and told apart from one whose body is an empty JSON value.
func TestAddInteraction_NoBody(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "interactions.json")

	for _, ia := range []*cassette.Interaction{
		interaction(t, nil, nil),
		interaction(t, []byte(`{}`), nil),
		interaction(t, nil, nil),
	} {
		if err := cassette.AddInteraction(path, ia); err != nil {
			t.Fatal(err)
		}
	}

	ias, err := cassette.InteractionsReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(ias) != 2 {
		t.Fatalf("held %d calls, want 2: one without a body and one with {}", len(ias))
	}

	if ias[0].Request.Body != nil || ias[0].Response.Body != nil {
		t.Errorf("a call without bodies read back with %q and %q", ias[0].Request.Body, ias[0].Response.Body)
	}
}
