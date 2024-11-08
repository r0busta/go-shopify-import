package importer

import (
	"io"

	"github.com/r0busta/go-shopify-graphql-model/v4/graph/model"
)

type ProductInput struct {
	Product  model.ProductInput               `json:"product,omitempty"`
	Variants []model.ProductVariantsBulkInput `json:"variants,omitempty"`
	Media    []model.CreateMediaInput         `json:"media,omitempty"`
}

type Decoder interface {
	Decode(io.Reader) ([]ProductInput, error)
}

func ToCreateInput(productInput model.ProductInput) model.ProductCreateInput {
	return model.ProductCreateInput{
		DescriptionHTML:        productInput.DescriptionHTML,
		Handle:                 productInput.Handle,
		Seo:                    productInput.Seo,
		ProductType:            productInput.ProductType,
		Category:               productInput.Category,
		Tags:                   productInput.Tags,
		TemplateSuffix:         productInput.TemplateSuffix,
		GiftCardTemplateSuffix: productInput.GiftCardTemplateSuffix,
		Title:                  productInput.Title,
		Vendor:                 productInput.Vendor,
		GiftCard:               productInput.GiftCard,
		CollectionsToJoin:      productInput.CollectionsToJoin,
		CombinedListingRole:    productInput.CombinedListingRole,
		Metafields:             productInput.Metafields,
		ProductOptions:         productInput.ProductOptions,
		Status:                 productInput.Status,
		RequiresSellingPlan:    productInput.RequiresSellingPlan,
		ClaimOwnership:         productInput.ClaimOwnership,
	}
}

func ToUpdateInput(productInput model.ProductInput) model.ProductUpdateInput {
	return model.ProductUpdateInput{
		DescriptionHTML:        productInput.DescriptionHTML,
		Handle:                 productInput.Handle,
		Seo:                    productInput.Seo,
		ProductType:            productInput.ProductType,
		Category:               productInput.Category,
		Tags:                   productInput.Tags,
		TemplateSuffix:         productInput.TemplateSuffix,
		GiftCardTemplateSuffix: productInput.GiftCardTemplateSuffix,
		Title:                  productInput.Title,
		Vendor:                 productInput.Vendor,
		RedirectNewHandle:      productInput.RedirectNewHandle,
		ID:                     productInput.ID,
		CollectionsToJoin:      productInput.CollectionsToJoin,
		CollectionsToLeave:     productInput.CollectionsToLeave,
		Metafields:             productInput.Metafields,
		Status:                 productInput.Status,
		RequiresSellingPlan:    productInput.RequiresSellingPlan,
	}
}
