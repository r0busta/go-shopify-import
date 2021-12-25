package importer_test

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/r0busta/go-shopify-graphql-model/v2/graph/model"
	"github.com/r0busta/go-shopify-graphql/v5"
	shopifymock "github.com/r0busta/go-shopify-graphql/v5/mock"
	"github.com/r0busta/go-shopify-import/v2/importer"
	graphqlmock "github.com/r0busta/graphql/mock"
	"github.com/stretchr/testify/require"
)

func TestDoOverwriteFalse(t *testing.T) {
	type args struct {
		data        string
		supplierTag string
		dedupBy     importer.DedupMode
	}
	tests := []struct {
		name                 string
		args                 args
		wantExistingProducts []model.Product
		wantProductCreate    []model.ProductInput
		wantProductUpdate    []model.ProductInput
		wantErr              bool
	}{
		{
			name: "empty data input",
			args: args{
				data:        `[]`,
				supplierTag: "supplier-tag",
				dedupBy:     importer.ComboDedupMode,
			},
			wantExistingProducts: []model.Product{},
		},
		{
			name: "create new products",
			args: args{
				data: `[
					{
						"handle": "handle-1",
						"variants": [{
							"sku": "sku-1-1"
						}]
					},
					{
						"handle": "handle-2",
						"variants": [{
							"sku": "sku-2-1"
						}]
					}
				]`,
				supplierTag: "supplier-tag",
				dedupBy:     importer.ComboDedupMode,
			},
			wantExistingProducts: []model.Product{},
			wantProductCreate: []model.ProductInput{
				{
					Handle: model.NewString("handle-1"),
					Variants: []model.ProductVariantInput{
						{
							Sku: model.NewString("sku-1-1"),
						},
					},
				},
				{
					Handle: model.NewString("handle-2"),
					Variants: []model.ProductVariantInput{
						{
							Sku: model.NewString("sku-2-1"),
						},
					},
				},
			},
		},
		{
			name: "don't overwrite existing products",
			args: args{
				data: `[
					{
						"handle": "handle-1",
						"variants": [{
							"sku": "sku-1-1"
						}]
					},
					{
						"handle": "handle-2",
						"variants": [{
							"sku": "sku-2-1"
						}]
					}
				]`,
				supplierTag: "supplier-tag",
				dedupBy:     importer.ComboDedupMode,
			},
			wantExistingProducts: []model.Product{
				{
					Handle: "handle-1",
					Variants: &model.ProductVariantConnection{
						Edges: []model.ProductVariantEdge{
							{
								Node: &model.ProductVariant{
									Sku: model.NewString("sku-1-1"),
								},
							},
						},
					},
				},
				{
					Handle: "handle-2",
					Variants: &model.ProductVariantConnection{
						Edges: []model.ProductVariantEdge{
							{
								Node: &model.ProductVariant{
									Sku: model.NewString("sku-2-1"),
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			productService := shopifymock.NewMockProductService(ctrl)

			shopClient := &shopify.Client{
				Product: productService,
			}

			jsonDecoder := newJSONDecoder()
			dataReader := strings.NewReader(tt.args.data)

			query := fmt.Sprintf(`tag:'%s'`, tt.args.supplierTag)
			productService.EXPECT().List(query).Return(tt.wantExistingProducts, nil)

			for _, p := range tt.wantProductCreate {
				productService.EXPECT().Create(structEq(p)).Return(nil, nil)
			}

			for _, p := range tt.wantProductUpdate {
				p := p
				productService.EXPECT().Update(structEq(p)).Return(nil)
			}

			const overwriteProducts = false
			const addMissingVariants = false
			err := importer.Do(shopClient, jsonDecoder, dataReader, tt.args.supplierTag, tt.args.dedupBy, overwriteProducts, addMissingVariants, nil, false)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDoOverwiteProducts(t *testing.T) {
	type args struct {
		supplierTag string
		dedupBy     importer.DedupMode
	}
	tests := []struct {
		name                 string
		args                 args
		importData           string
		wantExistingProducts []model.Product
		wantProductCreate    []model.ProductInput
		wantProductUpdate    []model.ProductInput
		wantErr              bool
	}{
		{
			name: "overwrites product data",
			args: args{
				supplierTag: "supplier-tag",
				dedupBy:     importer.ComboDedupMode,
			},
			importData: `[
				{
					"handle": "handle-1",
					"title": "title-1-new",
					"variants": [{
						"sku": "sku-1-1"
					},{
						"sku": "sku-1-2"
					},{
						"sku": "sku-1-3"
					},{
						"sku": "sku-1-4"
					}]
				},
				{
					"handle": "handle-2",
					"title": "title-2-new",
					"variants": [{
						"sku": "sku-2-1"
					},{
						"sku": "sku-2-2"
					}]
				}
			]`,
			wantExistingProducts: []model.Product{
				{
					ID:     "product-1",
					Handle: "handle-1",
					Title:  "title-1",
					Variants: &model.ProductVariantConnection{
						Edges: []model.ProductVariantEdge{
							{
								Node: &model.ProductVariant{
									ID:  "variant-1-1",
									Sku: model.NewString("sku-1-1"),
								},
							},
							{
								Node: &model.ProductVariant{
									ID:  "variant-1-3",
									Sku: model.NewString("sku-1-3"),
								},
							},
						},
					},
				},
				{
					ID:     "product-2",
					Handle: "handle-2",
					Title:  "title-2",
					Variants: &model.ProductVariantConnection{
						Edges: []model.ProductVariantEdge{
							{
								Node: &model.ProductVariant{
									ID:  "variant-2-1",
									Sku: model.NewString("sku-2-1"),
								},
							},
						},
					},
				},
				{
					ID:     "product-3",
					Handle: "handle-3",
					Title:  "title-3",
					Variants: &model.ProductVariantConnection{
						Edges: []model.ProductVariantEdge{
							{
								Node: &model.ProductVariant{
									ID:  "variant-3-1",
									Sku: model.NewString("sku-3-1"),
								},
							},
						},
					},
				},
			},
			wantProductUpdate: []model.ProductInput{
				{
					ID:     model.NewString("product-1"),
					Handle: model.NewString("handle-1"),
					Variants: []model.ProductVariantInput{
						{
							ID:  model.NewString("variant-1-1"),
							Sku: model.NewString("sku-1-1"),
						},
						{
							Sku: model.NewString("sku-1-2"),
						},
						{
							ID:  model.NewString("variant-1-3"),
							Sku: model.NewString("sku-1-3"),
						},
						{
							Sku: model.NewString("sku-1-4"),
						},
					},
				},
				{
					ID:     model.NewString("product-2"),
					Handle: model.NewString("handle-2"),
					Variants: []model.ProductVariantInput{
						{
							ID:  model.NewString("variant-2-1"),
							Sku: model.NewString("sku-2-1"),
						},
						{
							Sku: model.NewString("sku-2-2"),
						},
					},
				},
			},
		},
		{
			name: "error on empty variants in existing products",
			args: args{
				supplierTag: "supplier-tag",
				dedupBy:     importer.ComboDedupMode,
			},
			importData: `[]`,
			wantExistingProducts: []model.Product{
				{
					ID:     "product-1",
					Handle: "handle-1",
					Title:  "title-1",
					Variants: &model.ProductVariantConnection{
						Edges: []model.ProductVariantEdge{},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "error on empty variants in new products",
			args: args{
				supplierTag: "supplier-tag",
				dedupBy:     importer.ComboDedupMode,
			},
			importData: `[
				{
					"handle": "handle-1",
					"title": "title-1-new",
					"variants": []
				}
			]`,
			wantExistingProducts: []model.Product{},
			wantErr:              true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			productService := shopifymock.NewMockProductService(ctrl)

			shopClient := &shopify.Client{
				Product: productService,
			}

			jsonDecoder := newJSONDecoder()
			dataReader := strings.NewReader(tt.importData)

			query := fmt.Sprintf(`tag:'%s'`, tt.args.supplierTag)
			productService.EXPECT().List(query).Return(tt.wantExistingProducts, nil)

			for _, p := range tt.wantProductCreate {
				productService.EXPECT().Create(structEq(p)).Return(nil, nil)
			}

			for _, p := range tt.wantProductUpdate {
				p := p
				productService.EXPECT().Update(structEq(p)).Return(nil)
			}

			const overwriteProducts = true
			const addMissingVariants = false
			err := importer.Do(shopClient, jsonDecoder, dataReader, tt.args.supplierTag, tt.args.dedupBy, overwriteProducts, addMissingVariants, nil, false)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

type variantBulkCreateInput struct {
	ProductID                string
	ProductVariantsBulkInput []model.ProductVariantsBulkInput
}

type variantBulkReorderInput struct {
	ProductID                   string
	ProductVariantPositionInput []model.ProductVariantPositionInput
}

func TestDoCreateMissingVariants(t *testing.T) {
	type args struct {
		supplierTag string
		dedupBy     importer.DedupMode
		overwrite   bool
	}
	tests := []struct {
		name                      string
		args                      args
		importData                string
		wantExistingProducts      []model.Product
		wantProductCreate         []model.ProductInput
		wantProductUpdate         []model.ProductInput
		wantVariantBulkCreateVars []map[string]interface{}
		wantErr                   bool
	}{
		{
			name: "creates missing variants",
			args: args{
				supplierTag: "supplier-tag",
				dedupBy:     importer.ComboDedupMode,
				overwrite:   false,
			},
			importData: `[
				{
					"handle": "handle-1",
					"title": "title-1-new",
					"variants": [{
						"sku": "sku-1-1"
					},{
						"sku": "sku-1-2"
					},{
						"sku": "sku-1-3"
					},{
						"sku": "sku-1-4"
					}]
				},
				{
					"handle": "handle-2",
					"title": "title-2-new",
					"variants": [{
						"sku": "sku-2-1"
					},{
						"sku": "sku-2-2"
					}]
				}
			]`,
			wantExistingProducts: []model.Product{
				{
					ID:     "product-1",
					Handle: "handle-1",
					Title:  "title-1",
					Variants: &model.ProductVariantConnection{
						Edges: []model.ProductVariantEdge{
							{
								Node: &model.ProductVariant{
									ID:  "variant-1-1",
									Sku: model.NewString("sku-1-1"),
								},
							},
							{
								Node: &model.ProductVariant{
									ID:  "variant-1-3",
									Sku: model.NewString("sku-1-3"),
								},
							},
						},
					},
				},
				{
					ID:     "product-2",
					Handle: "handle-2",
					Title:  "title-2",
					Variants: &model.ProductVariantConnection{
						Edges: []model.ProductVariantEdge{
							{
								Node: &model.ProductVariant{
									ID:  "variant-2-1",
									Sku: model.NewString("sku-2-1"),
								},
							},
						},
					},
				},
				{
					ID:     "product-3",
					Handle: "handle-3",
					Title:  "title-3",
					Variants: &model.ProductVariantConnection{
						Edges: []model.ProductVariantEdge{
							{
								Node: &model.ProductVariant{
									ID:  "variant-3-1",
									Sku: model.NewString("sku-3-1"),
								},
							},
						},
					},
				},
			},
			wantVariantBulkCreateVars: []map[string]interface{}{
				{
					"productId": "product-1",
					"variants": []model.ProductVariantsBulkInput{
						{
							Sku: model.NewString("sku-1-2"),
						},
						{
							Sku: model.NewString("sku-1-4"),
						},
					},
				},
				{
					"productId": "product-2",
					"variants": []model.ProductVariantsBulkInput{
						{
							Sku: model.NewString("sku-2-2"),
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			gql := graphqlmock.NewMockGraphQL(ctrl)
			productService := shopifymock.NewMockProductService(ctrl)

			shopClient := shopify.NewClient("", "", "", shopify.WithGraphQLClient(gql))
			shopClient.Product = productService

			jsonDecoder := newJSONDecoder()
			dataReader := strings.NewReader(tt.importData)

			query := fmt.Sprintf(`tag:'%s'`, tt.args.supplierTag)
			productService.EXPECT().List(query).Return(tt.wantExistingProducts, nil)

			for _, p := range tt.wantProductCreate {
				productService.EXPECT().Create(structEq(p)).Return(nil, nil)
			}

			for _, p := range tt.wantProductUpdate {
				p := p
				productService.EXPECT().Update(structEq(p)).Return(nil)
			}

			for _, v := range tt.wantVariantBulkCreateVars {
				gql.EXPECT().Mutate(gomock.Any(), gomock.Any(), structEq(v)).Return(nil)
			}

			const overwriteProducts = false
			const addMissingVariants = true
			err := importer.Do(shopClient, jsonDecoder, dataReader, tt.args.supplierTag, tt.args.dedupBy, overwriteProducts, addMissingVariants, nil, false)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

type jsonDecoder struct{}

func newJSONDecoder() jsonDecoder {
	return jsonDecoder{}
}

func (j jsonDecoder) Decode(data io.Reader) ([]model.ProductInput, error) {
	products := []model.ProductInput{}
	err := json.NewDecoder(data).Decode(&products)
	if err != nil {
		return nil, fmt.Errorf("decoding: %w", err)
	}

	return products, nil
}

func structEq(v interface{}) gomock.Matcher {
	return gomock.GotFormatterAdapter(
		gomock.GotFormatterFunc(func(i interface{}) string {
			b, _ := json.MarshalIndent(i, "", "    ")

			return string(b)
		}),
		gomock.WantFormatter(
			gomock.StringerFunc(func() string {
				b, _ := json.MarshalIndent(v, "", "    ")

				return string(b)
			}),
			gomock.Eq(v),
		),
	)
}
