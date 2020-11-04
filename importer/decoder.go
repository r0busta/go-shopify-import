package importer

import (
	"io"

	shopifygraphql "github.com/r0busta/go-shopify-graphql"
)

type Decoder interface {
	Decode(io.Reader) ([]*shopifygraphql.ProductCreate, error)
}
