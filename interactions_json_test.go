package cassette_test

import (
	"bytes"
	"encoding/json/jsontext"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

// TestAddInteraction_MaskOptions redacts a header and a query parameter an API names to carry its key, which the
// default masking does not take for credentials, only where an option names them.
func TestAddInteraction_MaskOptions(t *testing.T) {
	t.Parallel()

	const key = "c3cr3tk3y"

	call := func() *cassette.Interaction {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://example.com/quote?symbol=F&code="+key, http.NoBody)
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Set("X-Finnhub", key)

		rec, err := cassette.NewRequest(req)
		if err != nil {
			t.Fatal(err)
		}

		return &cassette.Interaction{Request: rec, Response: cassette.Response{StatusCode: http.StatusOK}}
	}

	written := func(opts ...cassette.MaskOption) string {
		path := filepath.Join(t.TempDir(), "interactions.json")
		if err := cassette.AddInteraction(path, call(), opts...); err != nil {
			t.Fatal(err)
		}

		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}

		return string(b)
	}

	if got := written(); !strings.Contains(got, key) {
		t.Fatal("the default masking redacted a header and a query parameter it does not know")
	}

	got := written(cassette.MaskHeaders("x-finnhub"), cassette.MaskKeys("code"))
	if strings.Contains(got, key) {
		t.Errorf("the key is written: %s", got)
	}

	if !strings.Contains(got, "symbol=F") {
		t.Errorf("a parameter no option names is masked: %s", got)
	}
}

// A body of text other than JSON, such as XML, is written as a JSON string and read back, and replayed, as the text it
// was: quotes and line breaks included.
func TestInteractionsReadFile_TextBody(t *testing.T) {
	t.Parallel()

	const body = "<FlexStatementResponse timestamp=\"27 November, 2026\">\n<Status>Success</Status>\n</FlexStatementResponse>"

	path := filepath.Join(t.TempDir(), "interactions.json")
	ia := interaction(t, nil, []byte(body))
	ia.Response.Headers = http.Header{"Content-Type": {"text/xml;charset=UTF-8"}}

	if err := cassette.AddInteraction(path, ia); err != nil {
		t.Fatal(err)
	}

	ias, err := cassette.InteractionsReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if got := string(ias[0].Response.Body); got != body {
		t.Fatalf("read back %q, want %q", got, body)
	}

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "https://example.com/things", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/json")

	rsp, err := cassette.Replay(ias).RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer rsp.Body.Close()

	got, err := io.ReadAll(rsp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(got) != body {
		t.Errorf("replayed %q, want %q", got, body)
	}
}
