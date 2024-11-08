package importer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"

	diskstore "github.com/r0busta/go-object-store/disk"
	"github.com/r0busta/go-shopify-graphql-model/v4/graph/model"
	"github.com/r0busta/go-shopify-graphql/v9"
	"github.com/thoas/go-funk"
)

var productCache *diskstore.Store

func FetchAllProducts(shopClient *shopify.Client, supplierTag string, productCachePath *string, refreshCache bool) ([]model.Product, error) {
	useCache := false
	if productCachePath != nil && *productCachePath != "" {
		productCache = diskstore.New(*productCachePath)
		useCache = productCache.FileExists()

		if refreshCache {
			log.Println("cache cleared")
			productCache.Write([]model.Product{})
		}
	}

	products := []model.Product{}
	if useCache && !refreshCache {
		err := productCache.Read(&products)
		if err != nil {
			return nil, fmt.Errorf("error reading orders from cache: %w", err)
		}

		if len(products) == 0 {
			return nil, fmt.Errorf("cache is empty")
		}

		log.Printf("products from the cache (%d products)", len(products))
	} else {
		log.Printf("fetching all products from Shopify")

		var err error
		products, err = shopClient.Product.List(context.Background(), fmt.Sprintf(`tag:'%s'`, supplierTag))
		if err != nil {
			return nil, fmt.Errorf("error loading existing products: %w", err)
		}
		log.Printf("products fetched (%d products)", len(products))

		if productCache != nil {
			err = productCache.Write(products)
			if err != nil {
				log.Printf("warning: writing to the cache error: %s", err)
			}
			log.Println("products cached", len(products))
		}
	}

	return products, nil
}

func processProducts(new []ProductInput, old []model.Product, dedup DedupMode, overwriteProducts bool, addMissingVariants bool) ([]ProductInput, []ProductInput, []variantBulkCreateInput, error) {
	switch dedup {
	case ComboDedupMode:
		return mergeProductsByHandleAndSKU(new, old, overwriteProducts, addMissingVariants)
	default:
		return nil, nil, nil, fmt.Errorf("not-implemented dedup type: %s", dedup)
	}
}

type productInfo struct {
	handle string
	skus   []string
}

type duplicateSKUInfo struct {
	sku           string
	handle        string
	anotherHandle string
}

func mergeProductsByHandleAndSKU(newProducts []ProductInput, oldProducts []model.Product, overwriteProducts bool, addMissingVariants bool) ([]ProductInput, []ProductInput, []variantBulkCreateInput, error) {
	toCreate := []ProductInput{}
	toUpdate := []ProductInput{}
	variantsToAdd := []variantBulkCreateInput{}

	oldProductHandleMap := map[string]model.Product{}
	oldSKUHandleMap := map[string]string{}
	oldSKUSetInfoMap := map[string]productInfo{}

	for _, oldProduct := range oldProducts {
		if oldProduct.Handle == "" {
			return nil, nil, nil, fmt.Errorf("existing product without handle found (id=%s)", oldProduct.ID)
		}
		oldProductHandleMap[oldProduct.Handle] = oldProduct

		if oldProduct.Variants == nil || len(oldProduct.Variants.Edges) == 0 {
			return nil, nil, nil, fmt.Errorf("existing product without variants found (id=%s); this case is not supported yet", oldProduct.ID)
		}

		skuSet := map[string]struct{}{}
		skus := []string{}

		for _, v := range oldProduct.Variants.Edges {
			if v.Node == nil {
				return nil, nil, nil, fmt.Errorf("existing product variant without data found (id=%s)", oldProduct.ID)
			}

			if v.Node.InventoryItem == nil || isZero(v.Node.InventoryItem.Sku) {
				return nil, nil, nil, fmt.Errorf("existing variant without sku found (product_id=%s, variant_id=%s)", oldProduct.ID, v.Node.ID)
			}

			oldSKUHandleMap[*v.Node.InventoryItem.Sku] = oldProduct.Handle

			if _, ok := skuSet[*v.Node.InventoryItem.Sku]; ok {
				return nil, nil, nil, fmt.Errorf("duplicate sku found in the old product (product_id=%s, sku=%s)", oldProduct.ID, zeroOrValue(v.Node.InventoryItem.Sku))
			}

			skuSet[*v.Node.InventoryItem.Sku] = struct{}{}
			skus = append(skus, *v.Node.InventoryItem.Sku)
		}

		sort.Strings(skus)
		oldSKUSetInfoMap[skuSetStringify(skus)] = productInfo{
			handle: oldProduct.Handle,
			skus:   skus,
		}
	}

	duplicateSKUs := []duplicateSKUInfo{}
	for _, newProduct := range newProducts {
		if isZero(newProduct.Product.Handle) {
			return nil, nil, nil, fmt.Errorf("handle is empty (title=%s)", zeroOrValue(newProduct.Product.Title))
		}

		if len(newProduct.Variants) == 0 {
			return nil, nil, nil, fmt.Errorf("product has no variants (title=%s, handle=%s)", zeroOrValue(newProduct.Product.Title), zeroOrValue(newProduct.Product.Handle))
		}

		skuSet := map[string]struct{}{}
		skus := []string{}

		for _, v := range newProduct.Variants {
			if isZero(v.InventoryItem.Sku) {
				return nil, nil, nil, fmt.Errorf("new variant without sku found (%s)", zeroOrValue(newProduct.Product.Title))
			}

			if _, ok := skuSet[*v.InventoryItem.Sku]; ok {
				return nil, nil, nil, fmt.Errorf("duplicate sku found in the new product (title=%s, handle=%s, sku=%s)", zeroOrValue(newProduct.Product.Title), zeroOrValue(newProduct.Product.Handle), zeroOrValue(v.InventoryItem.Sku))
			}

			skuSet[*v.InventoryItem.Sku] = struct{}{}
			skus = append(skus, *v.InventoryItem.Sku)
		}

		sort.Strings(skus)

		var oldHandle *string
		var ok bool
		if oldHandle, ok = matchSKUSetPartially(oldSKUSetInfoMap, skus); !ok {
			for _, v := range newProduct.Variants {
				if anotherHandle, ok := oldSKUHandleMap[*v.InventoryItem.Sku]; ok && anotherHandle != *newProduct.Product.Handle {
					duplicateSKUs = append(duplicateSKUs, duplicateSKUInfo{
						sku:           *v.InventoryItem.Sku,
						handle:        *newProduct.Product.Handle,
						anotherHandle: anotherHandle,
					})
				}
			}

			continue
		}

		if *newProduct.Product.Handle != *oldHandle {
			log.Printf("matched product by the set of skus; keeping the old handle (old_handle=%s, new_handle=%s)", zeroOrValue(oldHandle), zeroOrValue(newProduct.Product.Handle))
			*newProduct.Product.Handle = *oldHandle
		}
	}

	if len(duplicateSKUs) > 0 {
		for _, d := range duplicateSKUs {
			log.Printf("trying to add a variant with the sku %s to %s that already exists in another product %s", d.sku, d.handle, d.anotherHandle)
		}
		return nil, nil, nil, fmt.Errorf("duplicate skus found (%d)", len(duplicateSKUs))
	}

	for _, newProduct := range newProducts {
		if isZero(newProduct.Product.Handle) {
			return nil, nil, nil, fmt.Errorf("handle is empty (title=%s)", zeroOrValue(newProduct.Product.Title))
		}
		if oldProduct, ok := oldProductHandleMap[*newProduct.Product.Handle]; ok {
			if overwriteProducts {
				newInput, err := mergeProductData(newProduct, oldProduct)
				if err != nil {
					return nil, nil, nil, fmt.Errorf("merging product data: %w", err)
				}
				toUpdate = append(toUpdate, *newInput)

				log.Printf("%s exists at id=%s — overwriting", zeroOrValue(newInput.Product.Handle), zeroOrValue(newInput.Product.ID))
			} else {
				if addMissingVariants {
					missingVariants := getMissingVariants(newProduct, oldProduct)
					if missingVariants != nil {
						log.Printf("adding %d variants to %s", len(missingVariants.ProductVariantsBulkInput), zeroOrValue(newProduct.Product.Handle))
						variantsToAdd = append(variantsToAdd, *missingVariants)
					}
				}

				// log.Printf("%s exists — skipping update", zeroOrValue(newProduct.Handle))

				continue
			}
		} else {
			log.Printf("%s will be created", zeroOrValue(newProduct.Product.Handle))
			toCreate = append(toCreate, newProduct)
		}
	}

	for _, variantToAdd := range variantsToAdd {
		for _, v := range variantToAdd.ProductVariantsBulkInput {
			if anotherHandle, ok := oldSKUHandleMap[*v.InventoryItem.Sku]; ok && anotherHandle != variantToAdd.ProductHandle {
				return nil, nil, nil, fmt.Errorf("trying to add a variant with the sku %s to %s that already exists in another product %s", *v.InventoryItem.Sku, variantToAdd.ProductHandle, anotherHandle)
			}
		}
	}

	return toCreate, toUpdate, variantsToAdd, nil
}

func mergeProductData(newData ProductInput, oldData model.Product) (*ProductInput, error) {
	variants, err := mergeVariants(newData.Variants, oldData.Variants.Edges)
	if err != nil {
		return nil, fmt.Errorf("merging variants: %w", err)
	}

	res := ProductInput{
		Product: model.ProductInput{
			ID:     model.NewString(oldData.ID),
			Handle: model.NewString(oldData.Handle),
		},
		Variants: variants,
	}

	return &res, nil
}

func getMissingVariants(newProduct ProductInput, oldProduct model.Product) *variantBulkCreateInput {
	if haveDifferentOptions(newProduct.Product.ProductOptions, oldProduct.Options) {
		log.Println("new product has different options", zeroOrValue(newProduct.Product.Handle), oldProduct.ID)

		return nil
	}

	oldSKUSet := map[string]struct{}{}
	for _, v := range oldProduct.Variants.Edges {
		oldSKUSet[*v.Node.InventoryItem.Sku] = struct{}{}
	}

	productVariantsBulkInput := []model.ProductVariantsBulkInput{}
	for _, newVariant := range newProduct.Variants {
		if _, ok := oldSKUSet[*newVariant.InventoryItem.Sku]; ok {
			continue
		}

		options := adjustOptionsOrder(newVariant.OptionValues, newProduct.Product.ProductOptions, oldProduct.Options)

		productVariantsBulkInput = append(productVariantsBulkInput, model.ProductVariantsBulkInput{
			ID:                  nil,
			Barcode:             newVariant.Barcode,
			CompareAtPrice:      newVariant.CompareAtPrice,
			MediaID:             newVariant.MediaID,
			MediaSrc:            newVariant.MediaSrc,
			InventoryPolicy:     newVariant.InventoryPolicy,
			InventoryQuantities: newVariant.InventoryQuantities,
			InventoryItem:       newVariant.InventoryItem,
			Metafields:          newVariant.Metafields,
			OptionValues:        options,
			Price:               newVariant.Price,
			Taxable:             newVariant.Taxable,
			TaxCode:             newVariant.TaxCode,
		})
	}

	if len(productVariantsBulkInput) > 0 {
		return &variantBulkCreateInput{
			ProductID:                oldProduct.ID,
			ProductHandle:            oldProduct.Handle,
			ProductVariantsBulkInput: productVariantsBulkInput,
		}
	}

	return nil
}

func haveDifferentOptions(newOptions []model.OptionCreateInput, oldOptions []model.ProductOption) bool {
	if len(newOptions) != len(oldOptions) {
		return true
	}

	options := []string{}
	for _, o := range oldOptions {
		options = append(options, o.Name)
	}

	for _, newOption := range newOptions {
		if !funk.ContainsString(options, *newOption.Name) {
			return true
		}
	}

	return false
}

func adjustOptionsOrder(selectedOptions []model.VariantOptionValueInput, newOptions []model.OptionCreateInput, oldOptions []model.ProductOption) []model.VariantOptionValueInput {
	if len(selectedOptions) != len(newOptions) {
		log.Panicln("selected options length is not equal to new options length")
	}

	if len(newOptions) == 0 {
		return nil
	}

	type positionStruct struct {
		position int
		id       string
	}
	positions := map[string]positionStruct{}
	for i, o := range oldOptions {
		positions[o.Name] = positionStruct{
			position: i,
			id:       o.ID,
		}
	}

	sortedOptions := make([]model.VariantOptionValueInput, len(newOptions))
	for i, newOption := range newOptions {
		newPosition := positions[*newOption.Name]
		sortedOptions[newPosition.position] = model.VariantOptionValueInput{
			ID:   model.NewString(newPosition.id),
			Name: model.NewString(*selectedOptions[i].Name),
		}
	}

	return sortedOptions
}

func createProductsBulk(s *shopify.Client, products []ProductInput) {
	for i, p := range products {
		log.Println(i+1, "of", len(products), "creating", zeroOrValue(p.Product.Handle))

		_, err := s.Product.Create(context.Background(), ToCreateInput(p.Product), p.Media)
		if err != nil {
			log.Printf("create product error: %s", err)
			b, _ := json.MarshalIndent(p, "", "    ")
			log.Println(string(b))
		}
	}
}

func updateProductsBulk(s *shopify.Client, products []ProductInput) {
	for i, p := range products {
		log.Println(i+1, "of", len(products), "updating", zeroOrValue(p.Product.Handle))

		err := s.Product.Update(context.Background(), ToUpdateInput(p.Product), p.Media)
		if err != nil {
			log.Printf("update product error: %s", err)
			b, _ := json.MarshalIndent(p, "", "    ")
			log.Println(string(b))
		}
	}
}

func skuSetStringify(skus []string) string {
	sort.Strings(skus)

	return strings.Join(skus, ",")
}

func matchSKUSetPartially(skuSetInfoMap map[string]productInfo, skus []string) (*string, bool) {
	if info, ok := skuSetInfoMap[skuSetStringify(skus)]; ok {
		return &info.handle, true
	}

	for _, info := range skuSetInfoMap {
		diff := funk.RightJoinString(info.skus, skus)
		if len(diff) == 0 || len(diff) < len(skus) {
			return &info.handle, true
		}
	}

	return nil, false
}
