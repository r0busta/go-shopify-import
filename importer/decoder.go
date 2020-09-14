package importer

import (
	"io"

	"github.com/r0busta/go-shopify-import/shop"
)

type Decoder interface {
	Decode(io.Reader) ([]*shop.ProductCreate, error)
}
