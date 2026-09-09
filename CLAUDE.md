# go-shopify-import

Library for importing products into a Shopify store from a supplier feed.
Module path `github.com/r0busta/go-shopify-import/v4`. No CLI.

## Layout

- `importer/import.go`: `Do`, the entry point. Decode, fetch existing,
  dedup, then create and update.
- `importer/decoder.go`: `Decoder` interface, `ProductInput`, and the
  `ToCreateInput` / `ToUpdateInput` converters.
- `importer/product.go`, `importer/variant.go`: fetching products by supplier
  tag (with a disk cache via go-object-store) and the bulk create/update
  calls.
- `importer/*_test.go`: gomock-driven tests using `go-shopify-graphql/mock`.
  They are the sanity check for library upgrades.

## Working here

- `go build ./... && go vet ./... && go test -race ./...` must pass.
- The importer mutates stores. Never run it against a store as a test.
- Depends on `go-shopify-graphql/v9` and `go-shopify-graphql-model/v4`.
  Bumping either major is a coordinated change.
