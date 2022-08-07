package importer

import (
	"io"

	"github.com/r0busta/go-shopify-graphql-model/v3/graph/model"
)

type Decoder interface {
	Decode(io.Reader) ([]model.ProductInput, error)
}
