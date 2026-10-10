---
tagline: Mock any Go HTTP client with the calls it really made.
---

Package cassette records HTTP calls as interactions, stored as JSON, and replays
them. One `http.RoundTripper`, `cassette.Replay`, does both: it mocks a client in
tests and records its real traffic, so there is no need to write one of your own.
