package importer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/r0busta/go-shopify-graphql-model/v4/graph/model"
	"github.com/r0busta/go-shopify-graphql/v9"
)

type variantBulkCreateInput struct {
	ProductID                string
	ProductHandle            string
	ProductVariantsBulkInput []model.ProductVariantsBulkInput
}

type variantBulkReorderInput struct {
	ProductID                   string
	ProductVariantPositionInput []model.ProductVariantPositionInput
}

type ProductOption struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Position int      `json:"position"`
	Values   []string `json:"values,omitempty"`
}

type VariantsBulkCreateProductResult struct {
	ID      string
	Handle  string
	Options []ProductOption
}

type VariantsBulkCreateVariantResult struct {
	ID              string
	Sku             string
	Position        int
	SelectedOptions []model.SelectedOption
	Product         *VariantsBulkCreateProductResult
}

type MutationProductVariantsBulkCreate struct {
	ProductVariantsBulkCreateResult ProductVariantsBulkCreateResult `graphql:"productVariantsBulkCreate(productId: $productId, variants: $variants)" json:"productVariantsBulkCreate"`
}

type ProductVariantsBulkCreateResult struct {
	Product         *VariantsBulkCreateProductResult  `json:"product,omitempty"`
	ProductVariants []VariantsBulkCreateVariantResult `json:"productVariants,omitempty"`

	UserErrors []model.UserError `json:"userErrors,omitempty"`
}

func createVariantsBulk(s *shopify.Client, bulkCreate []variantBulkCreateInput) error {
	for i, p := range bulkCreate {
		log.Println(i+1, "of", len(bulkCreate), "adding variants into the product", p.ProductID)
		err := s.Product.VariantsBulkCreate(context.Background(), p.ProductID, p.ProductVariantsBulkInput, model.ProductVariantsBulkCreateStrategyRemoveStandaloneVariant)
		if err != nil {
			log.Printf("bulk create variants: %s", err)
			b, _ := json.MarshalIndent(p, "", "    ")
			log.Println(string(b))
			return err
		}
	}

	return nil
}

func reorderVariantsBulk(s *shopify.Client, bulkReorder []variantBulkReorderInput) {
	for i, p := range bulkReorder {
		log.Println(i+1, "of", len(bulkReorder), "reordering variants in the product", p.ProductID)
		err := s.Product.VariantsBulkReorder(context.Background(), p.ProductID, p.ProductVariantPositionInput)
		if err != nil {
			log.Printf("bulk reorder variants: %s", err)
			b, _ := json.MarshalIndent(p, "", "    ")
			log.Println(string(b))
		}
	}
}

func mergeVariants(newVariants []model.ProductVariantsBulkInput, oldVariants []model.ProductVariantEdge) ([]model.ProductVariantsBulkInput, error) {
	variants := []model.ProductVariantsBulkInput{}

	oldVariantSKUMap := map[string]*model.ProductVariant{}
	for _, variant := range oldVariants {
		if variant.Node == nil {
			return nil, fmt.Errorf("existing variant without data found")
		}

		if isZero(variant.Node.InventoryItem.Sku) {
			return nil, fmt.Errorf("existing variant without sku found")
		}

		oldVariantSKUMap[*variant.Node.InventoryItem.Sku] = variant.Node
	}

	for _, variant := range newVariants {
		if isZero(variant.InventoryItem.Sku) {
			return nil, fmt.Errorf("new variant without sku found")
		}

		if existing, ok := oldVariantSKUMap[*variant.InventoryItem.Sku]; ok {
			newVariant, err := mergeVariantData(variant, existing)
			if err != nil {
				return nil, fmt.Errorf("merging variant data: %w", err)
			}
			variants = append(variants, newVariant)
		} else {
			variants = append(variants, variant)
		}
	}

	return variants, nil
}

func mergeVariantData(newData model.ProductVariantsBulkInput, oldData *model.ProductVariant) (model.ProductVariantsBulkInput, error) {
	res := newData
	res.ID = model.NewString(oldData.ID)

	return res, nil
}
