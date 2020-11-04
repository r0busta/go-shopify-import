package importer

import (
	"io"

	shop "github.com/r0busta/go-shopify-graphql"
)

type Decoder interface {
	Decode(io.Reader) ([]*shop.ProductCreate, error)
}
