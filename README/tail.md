## Where it comes from

cassette started inside
[openapi-enrich](https://github.com/MarkRosemaker/openapi-enrich), which turns
recorded traffic into an OpenAPI specification, and
[openapi-codegen](https://github.com/MarkRosemaker/openapi-codegen) generates
clients whose tests replay it. It moved into a module of its own so that those
clients, and anyone else, can record and replay without depending on either.
