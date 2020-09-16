package shop

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"

	"github.com/shurcooL/graphql"
)

type ProductService interface {
	Create(product *ProductCreate) error
	CreateBulk(products []*ProductCreate) error
}

type ProductServiceOp struct {
	client *Client
}

type ProductCreate struct {
	ProductInput ProductInput
	MediaInput   []CreateMediaInput
}

type ProductInput struct {
	// The IDs of the collections that this product will be added to.
	CollectionsToJoin []graphql.ID `json:"collectionsToJoin,omitempty"`

	// The IDs of collections that will no longer include the product.
	CollectionsToLeave []graphql.ID `json:"collectionsToLeave,omitempty"`

	// The description of the product, complete with HTML formatting.
	DescriptionHTML graphql.String `json:"descriptionHtml,omitempty"`

	// Whether the product is a gift card.
	GiftCard graphql.Boolean `json:"giftCard,omitempty"`

	// The theme template used when viewing the gift card in a store.
	GiftCardTemplateSuffix graphql.String `json:"giftCardTemplateSuffix,omitempty"`

	// A unique human-friendly string for the product. Automatically generated from the product's title.
	Handle graphql.String `json:"handle,omitempty"`

	// Specifies the product to update in productUpdate or creates a new product if absent in productCreate.
	ID graphql.ID `json:"id,omitempty"`

	// The images to associate with the product.
	Images []ImageInput `json:"images,omitempty"`

	// The metafields to associate with this product.
	Metafields []MetafieldInput `json:"metafields,omitempty"`

	// List of custom product options (maximum of 3 per product).
	Options []graphql.String `json:"options,omitempty"`

	// The private metafields to associated with this product.
	PrivateMetafields []PrivateMetafieldInput `json:"privateMetafields,omitempty"`

	// The product type specified by the merchant.
	ProductType graphql.String `json:"productType,omitempty"`

	// Whether a redirect is required after a new handle has been provided. If true, then the old handle is redirected to the new one automatically.
	RedirectNewHandle graphql.Boolean `json:"redirectNewHandle,omitempty"`

	// The SEO information associated with the product.
	SEO *SEOInput `json:"seo,omitempty"`

	// A comma separated list tags that have been added to the product.
	Tags []graphql.String `json:"tags,omitempty"`

	// The theme template used when viewing the product in a store.
	TemplateSuffix graphql.String `json:"templateSuffix,omitempty"`

	// The title of the product.
	Title graphql.String `json:"title,omitempty"`

	// A list of variants associated with the product.
	Variants []ProductVariantInput `json:"variants,omitempty"`

	// The name of the product's vendor.
	Vendor graphql.String `json:"vendor,omitempty"`
}

type CreateMediaInput struct {
	Alt              graphql.String   `json:"alt,omitempty"`
	MediaContentType MediaContentType `json:"mediaContentType,omitempty"` // REQUIRED
	OriginalSource   graphql.String   `json:"originalSource,omitempty"`   // REQUIRED
}

type MediaContentType string // Enum of strings: EXTERNAL_VIDEO, IMAGE, MODEL_3D, VIDEO

type MetafieldInput struct {
	ID        graphql.ID         `json:"id,omitempty"`
	Namespace graphql.String     `json:"namespace,omitempty"`
	Key       graphql.String     `json:"key,omitempty"`
	Value     graphql.String     `json:"value,omitempty"`
	ValueType MetafieldValueType `json:"valueType,omitempty"`
}

type PrivateMetafieldInput struct {
	Key        graphql.String              `json:"key,omitempty"`       // REQUIRED
	Namespace  graphql.String              `json:"namespace,omitempty"` // REQUIRED
	Owner      graphql.ID                  `json:"owner,omitempty"`
	ValueInput *PrivateMetafieldValueInput `json:"valueInput,omitempty"` // REQUIRED
}

type PrivateMetafieldValueInput struct {
	Value     graphql.String            `json:"value,omitempty"`     // REQUIRED
	ValueType PrivateMetafieldValueType `json:"valueType,omitempty"` // REQUIRED
}

type PrivateMetafieldValueType string // Enum of strings: INTEGER, JSON_STRING, STRING

type MetafieldValueType string // Enum of strings: INTEGER, JSON_STRING, STRING

type SEOInput struct {
	Description graphql.String `json:"description,omitempty"`
	Title       graphql.String `json:"title,omitempty"`
}

type ImageInput struct {
	AltText graphql.String `json:"altText,omitempty"`
	ID      graphql.ID     `json:"id,omitempty"`
	Src     graphql.String `json:"src,omitempty"`
}

type mutationProductCreate struct {
	ProductCreateResult productCreateResult `graphql:"productCreate(input: $input, media: $media)"`
}

type productCreateResult struct {
	Product struct {
		ID string `json:"id,omitempty"`
	}
	UserErrors []UserErrors
}

func (s *ProductServiceOp) CreateBulk(products []*ProductCreate) error {
	for _, p := range products {
		err := s.Create(p)
		if err != nil {
			log.Println("Warning! Couldn't create product (%v)", p)
		}
	}

	return nil
}

func WriteFormatedJSON(w io.Writer, v interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "    ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

func (s *ProductServiceOp) Create(product *ProductCreate) error {
	m := mutationProductCreate{}

	vars := map[string]interface{}{
		"input": product.ProductInput,
		"media": product.MediaInput,
	}
	err := s.client.gql.Mutate(context.Background(), &m, vars)
	if err != nil {
		return err
	}

	if len(m.ProductCreateResult.UserErrors) > 0 {
		return fmt.Errorf("%+v", m.ProductCreateResult.UserErrors)
	}

	return nil
}
