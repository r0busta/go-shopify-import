package importer

import (
	"io"

	"github.com/r0busta/go-shopify-graphql/v2"
)

type Decoder interface {
	Decode(io.Reader) ([]*shopify.ProductCreate, error)
}
