package importer

import (
	"fmt"
	"io"
	"log"
	"os"

	diskstore "github.com/r0busta/go-object-store/disk"
	"github.com/r0busta/go-shopify-graphql/v5"
)

type DedupMode string

const (
	ComboDedupMode DedupMode = "combo"
)

func Do(shopClient *shopify.Client, decoder Decoder, importData io.Reader, supplierTag string, dedup DedupMode, overwriteProducts bool, addMissingVariants bool, productCachePath *string, refreshCache bool) error {
	importProducts, err := decoder.Decode(importData)
	if err != nil {
		return fmt.Errorf("error decoding data: %w", err)
	}
	log.Printf("data parsed (%d products)", len(importProducts))

	existingProducts, err := fetchAllProducts(shopClient, supplierTag, productCachePath, refreshCache)
	if err != nil {
		return fmt.Errorf("fetching products: %s", err)
	}

	productsCreate, productsUpdate, variantsCreate, err := processProducts(importProducts, existingProducts, dedup, overwriteProducts, addMissingVariants)
	if err != nil {
		return fmt.Errorf("deduplicate products: %w", err)
	}

	if (len(productsCreate) > 0 || len(productsUpdate) > 0 || len(variantsCreate) > 0) && productCachePath != nil {
		productCache = diskstore.New(*productCachePath)
		if productCache.FileExists() {
			err := os.Remove(*productCachePath)
			if err != nil {
				log.Panicln("error removing cache file:", err)
			}
			log.Println("cache removed")
		}
	}

	log.Printf("importing products: %d to be created and %d to be updated", len(productsCreate), len(productsUpdate))

	createProductsBulk(shopClient, productsCreate)
	updateProductsBulk(shopClient, productsUpdate)

	log.Printf("updating variants: %d to be created", len(variantsCreate))
	createVariantsBulk(shopClient, variantsCreate)

	return nil
}
