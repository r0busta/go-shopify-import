# Contributing

This library was built for one shop and is published as-is, so expect rough
edges. Pull requests are welcome.

- Run `gofmt`, `go vet ./...` and `go test -race ./...`. CI runs the same.
- Keep the module path as `github.com/r0busta/go-shopify-import/v4`.
- Tests use gomock mocks from `go-shopify-graphql/mock`; if a change needs a
  service method that is not mocked yet, the mock lives in that repo.
- Never point a test or example at a real store. The importer creates and
  updates products.
- Version chain: this module pins a major version of go-shopify-graphql and
  go-shopify-graphql-model. A schema bump in the model means new majors of
  both libraries, and then of this module.
