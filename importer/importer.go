package importer

import (
	"fmt"
	"os"

	"github.com/r0busta/go-shopify-import/shop"
)

const (
	CSV = ".csv"
)

func Do(shopClient *shop.Client, decoder Decoder, locationID int64, inputFile string) error {
	f, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("error opening input file: %s", err)
	}
	defer f.Close()

	products, err := decoder.Decode(f)
	if err != nil {
		return fmt.Errorf("error decoding input file: %s", err)
	}

	err = shopClient.Product.CreateBulk(products)
	if err != nil {
		return fmt.Errorf("error creating products: %s", err)
	}

	return nil
}
