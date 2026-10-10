- **Mock** a client in tests with `cassette.Replay(ias)`: each request is
  answered from the recording it matches, by method, URL with its query in any
  order, and body, compared as JSON. Any other request fails with
  `cassette.ErrNoInteraction`, without touching the network.
- **Record** with `cassette.Replay(ias, cassette.RecordMisses(nil))`: a request
  without a recording is sent and recorded, and `Recorder.Interactions` holds
  every call, ready for `Interactions.WriteFile`.
- **Test strictly** with `cassette.InOrder()` and `cassette.CompareHeaders(...)`:
  every call must match the next recording, headers included, and
  `Recorder.Unused` tells which recordings were never called.
- **Masking**: writing a file always masks credentials first, in headers, bodies
  and query values. `Masker` says which, and `Masker.Keep` spares a header.
- **Once per call**: `Masker.AddInteraction` writes a call to a file only if it is not
  in it yet, so a failure that recurs is recorded once.
- **Large bodies**: a body that is not text is never read or written, only
  marked `bodyOmitted`, so a download streams to its caller as it is. A JSON
  string longer than 2,048 bytes, such as an image in base64, is cut to that.
- **Stable files**: query parameters are sorted by name, so a recording does not
  change with the order a client happens to send them in.
