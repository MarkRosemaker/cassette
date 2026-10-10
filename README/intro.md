A cassette is an `interactions.json` file: a list of requests, each with the
response it got. Hand it to `cassette.Replay` and any client that takes an
`*http.Client` gets those responses, without touching the network.

The package depends on nothing but the standard library and
[jsonutil](https://github.com/MarkRosemaker/jsonutil), so a library can depend
on it, and offer it to its users, at no cost.
