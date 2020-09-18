package importer

import (
	"fmt"
	"log"
	"os"

	"github.com/r0busta/go-shopify-import/shop"

	"github.com/shurcooL/graphql"
)

const (
	CSV = ".csv"
)

func Do(shopClient *shop.Client, decoder Decoder, inputFile, dedupBy string, overwrite bool, status shop.ProgressStatus) error {
	f, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("error opening input file: %s", err)
	}
	defer f.Close()

	products, err := decoder.Decode(f)
	if err != nil {
		return fmt.Errorf("error decoding input file: %s", err)
	}
	log.Println("Data feed parsed")

	existing, err := shopClient.Product.List()
	if err != nil {
		return fmt.Errorf("error loading existing products: %s", err)
	}

	toCreate, toUpdate := dedupProducts(products, existing, dedupBy, overwrite)

	log.Printf("Importing products: %d to be created and %d to be updated", len(toCreate), len(toUpdate))

	err = shopClient.Product.CreateBulk(toCreate, status)
	if err != nil {
		return fmt.Errorf("error creating products: %s", err)
	}

	err = shopClient.Product.UpdateBulk(toUpdate, status)
	if err != nil {
		return fmt.Errorf("error creating products: %s", err)
	}

	return nil
}

func dedupProducts(new []*shop.ProductCreate, old []*shop.Product, dedupBy string, overwrite bool) ([]*shop.ProductCreate, []*shop.ProductUpdate) {
	toCreate := []*shop.ProductCreate{}
	toUpdate := []*shop.ProductUpdate{}

	if dedupBy == "handle" {
		lookup := map[graphql.String]*shop.Product{}
		for _, p := range old {
			lookup[p.Handle] = p
		}

		for _, p := range new {
			if p.ProductInput.Handle == graphql.String("") {
				log.Fatalln("Handle is empty", p.ProductInput.Title)
			}
			if existing, ok := lookup[p.ProductInput.Handle]; ok {
				if overwrite {
					copyInput := p.ProductInput
					copyInput.ID = existing.ID
					update := &shop.ProductUpdate{ProductInput: copyInput}

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
	} else {
		log.Fatalln("Not-implemented dedup type", dedupBy)
		return []*shop.ProductCreate{}, []*shop.ProductUpdate{}
	}

	return toCreate, toUpdate
}
