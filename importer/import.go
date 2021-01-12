package importer

import (
	"fmt"
	"log"
	"os"

	"github.com/r0busta/go-shopify-graphql/v2"
	"github.com/r0busta/graphql"
	"github.com/thoas/go-funk"
)

func Do(shopClient *shopify.Client, decoder Decoder, inputFile, supplierTag, dedupBy string, overwrite bool) error {
	f, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("error opening input file: %s", err)
	}
	defer f.Close()

	products, err := decoder.Decode(f)
	if err != nil {
		return fmt.Errorf("error decoding input file: %s", err)
	}
	log.Printf("Data feed parsed (%d products)", len(products))

	os.Exit(0)

	existing, err := shopClient.Product.List(fmt.Sprintf(`tag:'%s'`, supplierTag))
	if err != nil {
		return fmt.Errorf("error loading existing products: %s", err)
	}
	log.Printf("Existing products retrieved (%d products)", len(existing))

	toCreate, toUpdate := dedupProducts(products, existing, dedupBy, overwrite)
	log.Printf("Importing products: %d to be created and %d to be updated", len(toCreate), len(toUpdate))

	err = shopClient.Product.CreateBulk(toCreate)
	if err != nil {
		return fmt.Errorf("error creating products: %s", err)
	}

	err = shopClient.Product.UpdateBulk(toUpdate)
	if err != nil {
		return fmt.Errorf("error creating products: %s", err)
	}

	return nil
}

func dedupProducts(new []*shopify.ProductCreate, old []*shopify.ProductBulkResult, dedupBy string, overwrite bool) (toCreate []*shopify.ProductCreate, toUpdate []*shopify.ProductUpdate) {
	switch dedupBy {
	case "handle":
		return dedupProductsByHandle(new, old, overwrite)
	case "sku":
		return dedupProductsBySKU(new, old, overwrite)
	default:
		log.Fatalln("Not-implemented dedup type", dedupBy)
		return []*shopify.ProductCreate{}, []*shopify.ProductUpdate{}
	}
}

func dedupProductsByHandle(new []*shopify.ProductCreate, old []*shopify.ProductBulkResult, overwrite bool) (toCreate []*shopify.ProductCreate, toUpdate []*shopify.ProductUpdate) {
	toCreate = []*shopify.ProductCreate{}
	toUpdate = []*shopify.ProductUpdate{}

	lookup := map[graphql.String]*shopify.ProductBulkResult{}
	for _, p := range old {
		lookup[p.Handle] = p
	}

	for _, p := range new {
		if p.ProductInput.Handle == graphql.String("") {
			log.Fatalln("Handle is empty", p.ProductInput.Title)
		}
		if existing, ok := lookup[p.ProductInput.Handle]; ok {
			if overwrite {
				newInput := mergeProductData(p.ProductInput, existing)
				update := &shopify.ProductUpdate{ProductInput: newInput}

				log.Printf("%s exists at %s. Overwriting.", update.ProductInput.Handle, update.ProductInput.ID)
				toUpdate = append(toUpdate, update)
			} else {
				log.Printf("%s exists. Skipping.", p.ProductInput.Handle)
				continue
			}
		} else {
			toCreate = append(toCreate, p)
		}
	}

	return
}

func dedupProductsBySKU(new []*shopify.ProductCreate, old []*shopify.ProductBulkResult, overwrite bool) (toCreate []*shopify.ProductCreate, toUpdate []*shopify.ProductUpdate) {
	toCreate = []*shopify.ProductCreate{}
	toUpdate = []*shopify.ProductUpdate{}

	allSKUs := []string{}
	lookup := map[string]*shopify.ProductBulkResult{}
	for _, p := range old {
		for _, v := range p.ProductVariants {
			lookup[string(v.SKU)] = p
			allSKUs = append(allSKUs, string(v.SKU))
		}
	}

	for _, p := range new {
		if len(p.ProductInput.Variants) == 0 {
			log.Fatalln("No variants", p.ProductInput.Title)
		}
		SKUs := []string{}
		for _, v := range p.ProductInput.Variants {
			if v.SKU == graphql.String("") {
				log.Fatalln("SKU is empty", p.ProductInput.Title, v.Options)
			}
			SKUs = append(SKUs, string(v.SKU))
		}

		newSKUs := funk.LeftJoinString(SKUs, allSKUs)
		if len(newSKUs) < len(SKUs) {
			var existing *shopify.ProductBulkResult
			for _, s := range SKUs {
				var ok bool
				if existing, ok = lookup[s]; ok {
					break
				}
			}
			if existing == nil {
				log.Fatalln("Matching product exists but no corresponding object found in the shop data")
			}
			if overwrite {
				newInput := mergeProductData(p.ProductInput, existing)
				update := &shopify.ProductUpdate{ProductInput: newInput}

				log.Printf("%s exists at %s. Overwriting.", update.ProductInput.Title, update.ProductInput.ID)
				toUpdate = append(toUpdate, update)
			} else {
				log.Printf("%s exists. Skipping.", p.ProductInput.Title)
				continue
			}
		} else {
			toCreate = append(toCreate, p)
		}
	}

	return
}

func mergeProductData(input shopify.ProductInput, data *shopify.ProductBulkResult) shopify.ProductInput {
	input.ID = data.ID
	for i := 0; i < len(input.Metafields); i++ {
		for _, oldM := range data.Metafields {
			if oldM.Namespace == input.Metafields[i].Namespace && oldM.Key == input.Metafields[i].Key {
				input.Metafields[i].ID = oldM.ID
			}
		}
	}

	return input
}
