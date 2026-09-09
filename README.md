# go-shopify-import

Import products into a Shopify store from a supplier feed (CSV, XML, or any
format you can write a decoder for), creating new products and updating or
completing existing ones.

This is a Go library, not a CLI. It was written for a specific shop and is
published as-is. It uses
[go-shopify-graphql](https://github.com/r0busta/go-shopify-graphql) and the
types from
[go-shopify-graphql-model](https://github.com/r0busta/go-shopify-graphql-model).

## What it does

`importer.Do` takes a decoder and a feed, then:

1. Decodes the feed into a list of `importer.ProductInput` (product input,
   its variants, and media).
2. Fetches every existing product carrying the supplier tag, optionally from
   a local cache file.
3. Deduplicates by product handle. Products that already exist are updated
   when `overwriteProducts` is set, and variants missing from an existing
   product are created when `addMissingVariants` is set.
4. Creates new products and their variants, and applies updates, using bulk
   variant mutations.

It writes to the store. Run it against a development store first.

## Usage

```go
package main

import (
	"os"

	shopify "github.com/r0busta/go-shopify-graphql/v9"
	"github.com/r0busta/go-shopify-import/v4/importer"
)

type csvDecoder struct{}

func (csvDecoder) Decode(r io.Reader) ([]importer.ProductInput, error) {
	// parse the feed and build model.ProductInput, model.ProductVariantsBulkInput
	// and model.CreateMediaInput values for each product
}

func main() {
	client := shopify.NewClientWithToken(os.Getenv("STORE_ACCESS_TOKEN"), os.Getenv("STORE_NAME"))

	feed, err := os.Open("products.csv")
	if err != nil {
		panic(err)
	}
	defer feed.Close()

	cache := "products-cache.json"
	err = importer.Do(client, csvDecoder{}, feed,
		"supplier:acme",          // tag that marks this supplier's products
		importer.ComboDedupMode,  // dedup mode
		false,                    // overwrite existing products
		true,                     // add variants missing from existing products
		&cache,                   // product cache path, nil to disable
		false,                    // refresh the cache
	)
	if err != nil {
		panic(err)
	}
}
```

`ToCreateInput` and `ToUpdateInput` convert a `model.ProductInput` into the
create and update input types the Admin API expects.

## Testing

```bash
go test ./...
```

The tests drive the importer through gomock mocks of the service interfaces
from go-shopify-graphql, so they need no store access.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).
