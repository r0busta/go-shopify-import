package importer

import (
	"testing"

	"github.com/r0busta/go-shopify-graphql-model/v2/graph/model"
	"github.com/stretchr/testify/assert"
)

func Test_dedupProductsByHandle_overwrite(t *testing.T) {
	type args struct {
		new []model.ProductInput
		old []model.Product
	}
	tests := []struct {
		name         string
		args         args
		wantToCreate []model.ProductInput
		wantToUpdate []model.ProductInput
		wantError    bool
	}{
		{
			name: "no products matching by handle — a product will be created",
			args: args{
				new: []model.ProductInput{
					{
						Handle: model.NewString("handle-1"),
						Variants: []model.ProductVariantInput{
							{
								Sku: model.NewString("sku-1"),
							},
						},
					},
				},
				old: []model.Product{
					{
						ID:     "2",
						Handle: "handle-2",
						Variants: &model.ProductVariantConnection{
							Edges: []model.ProductVariantEdge{
								{
									Node: &model.ProductVariant{
										Sku: model.NewString("sku-2"),
									},
								},
							},
						},
					},
				},
			},
			wantToCreate: []model.ProductInput{
				{
					Handle: model.NewString("handle-1"),
					Variants: []model.ProductVariantInput{
						{
							Sku: model.NewString("sku-1"),
						},
					},
				},
			},
			wantToUpdate: []model.ProductInput{},
		},
		{
			name: "products match by handle will be overwritten",
			args: args{
				new: []model.ProductInput{
					{
						Handle: model.NewString("handle-1"),
						Variants: []model.ProductVariantInput{
							{
								Sku: model.NewString("sku-1"),
							},
						},
					},
				},
				old: []model.Product{
					{
						ID:     "1",
						Handle: "handle-1",
						Variants: &model.ProductVariantConnection{
							Edges: []model.ProductVariantEdge{
								{
									Node: &model.ProductVariant{
										Sku: model.NewString("sku-2"),
									},
								},
							},
						},
					},
				},
			},
			wantToCreate: []model.ProductInput{},
			wantToUpdate: []model.ProductInput{
				{
					ID:     model.NewString("1"),
					Handle: model.NewString("handle-1"),
					Variants: []model.ProductVariantInput{
						{
							Sku: model.NewString("sku-1"),
						},
					},
				},
			},
		},
		{
			name: "both create and overwrite cases",
			args: args{
				new: []model.ProductInput{
					{
						Handle: model.NewString("handle-2"),
						Variants: []model.ProductVariantInput{
							{
								Sku: model.NewString("sku-2"),
							},
						},
					},
					{
						Handle: model.NewString("handle-1"),
						Variants: []model.ProductVariantInput{
							{
								Sku: model.NewString("sku-1"),
							},
						},
					},
				},
				old: []model.Product{
					{
						ID:     "1",
						Handle: "handle-1",
						Variants: &model.ProductVariantConnection{
							Edges: []model.ProductVariantEdge{
								{
									Node: &model.ProductVariant{
										Sku: model.NewString("sku-3"),
									},
								},
							},
						},
					},
				},
			},
			wantToCreate: []model.ProductInput{
				{
					Handle: model.NewString("handle-2"),
					Variants: []model.ProductVariantInput{
						{
							Sku: model.NewString("sku-2"),
						},
					},
				},
			},
			wantToUpdate: []model.ProductInput{
				{
					ID:     model.NewString("1"),
					Handle: model.NewString("handle-1"),
					Variants: []model.ProductVariantInput{
						{
							Sku: model.NewString("sku-1"),
						},
					},
				},
			},
		},
		{
			name: "matches by the variants' set of sku too",
			args: args{
				new: []model.ProductInput{
					{
						Handle: model.NewString("handle-1-new"),
						Variants: []model.ProductVariantInput{
							{
								Sku: model.NewString("sku-1"),
							},
							{
								Sku: model.NewString("sku-2"),
							},
						},
					},
				},
				old: []model.Product{
					{
						ID:     "1",
						Handle: "handle-1",
						Variants: &model.ProductVariantConnection{
							Edges: []model.ProductVariantEdge{
								{
									Node: &model.ProductVariant{
										ID:  "variant-1",
										Sku: model.NewString("sku-1"),
									},
								},
								{
									Node: &model.ProductVariant{
										ID:  "variant-2",
										Sku: model.NewString("sku-2"),
									},
								},
							},
						},
					},
				},
			},
			wantToCreate: []model.ProductInput{},
			wantToUpdate: []model.ProductInput{
				{
					ID:     model.NewString("1"),
					Handle: model.NewString("handle-1"),
					Variants: []model.ProductVariantInput{
						{
							ID:  model.NewString("variant-1"),
							Sku: model.NewString("sku-1"),
						},
						{
							ID:  model.NewString("variant-2"),
							Sku: model.NewString("sku-2"),
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

			const overwriteProducts = true
			const addMissingVariants = false
			gotToCreate, gotToUpdate, _, err := mergeProductsByHandleAndSKU(tt.args.new, tt.args.old, overwriteProducts, addMissingVariants)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.wantToCreate, gotToCreate)
			assert.Equal(t, tt.wantToUpdate, gotToUpdate)
		})
	}
}

func Test_dedupProductsByHandle_do_not_overwrite_and_do_not_create_variants(t *testing.T) {
	type args struct {
		new []model.ProductInput
		old []model.Product
	}
	tests := []struct {
		name                 string
		args                 args
		wantToCreate         []model.ProductInput
		wantToUpdate         []model.ProductInput
		wantToCreateVariants []variantBulkCreateInput
		wantError            bool
	}{
		{
			name: "no products matching by handle — a product will be created",
			args: args{
				new: []model.ProductInput{
					{
						Handle: model.NewString("handle-1"),
						Variants: []model.ProductVariantInput{
							{
								Sku: model.NewString("sku-1"),
							},
						},
					},
				},
				old: []model.Product{
					{
						ID:     "2",
						Handle: "handle-2",
						Variants: &model.ProductVariantConnection{
							Edges: []model.ProductVariantEdge{
								{
									Node: &model.ProductVariant{
										Sku: model.NewString("sku-2"),
									},
								},
							},
						},
					},
				},
			},
			wantToCreate: []model.ProductInput{
				{
					Handle: model.NewString("handle-1"),
					Variants: []model.ProductVariantInput{
						{
							Sku: model.NewString("sku-1"),
						},
					},
				},
			},
			wantToUpdate:         []model.ProductInput{},
			wantToCreateVariants: []variantBulkCreateInput{},
		},
		{
			name: "products match by handle won't be overwritten",
			args: args{
				new: []model.ProductInput{
					{
						Handle: model.NewString("handle-1"),
						Variants: []model.ProductVariantInput{
							{
								Sku: model.NewString("sku-1"),
							},
						},
					},
				},
				old: []model.Product{
					{
						ID:     "1",
						Handle: "handle-1",
						Variants: &model.ProductVariantConnection{
							Edges: []model.ProductVariantEdge{
								{
									Node: &model.ProductVariant{
										Sku: model.NewString("sku-2"),
									},
								},
							},
						},
					},
				},
			},
			wantToCreate:         []model.ProductInput{},
			wantToUpdate:         []model.ProductInput{},
			wantToCreateVariants: []variantBulkCreateInput{},
		},
		{
			name: "both create and don't overwrite cases",
			args: args{
				new: []model.ProductInput{
					{
						Handle: model.NewString("handle-2"),
						Variants: []model.ProductVariantInput{
							{
								Sku: model.NewString("sku-2"),
							},
						},
					},
					{
						Handle: model.NewString("handle-1"),
						Variants: []model.ProductVariantInput{
							{
								Sku: model.NewString("sku-1"),
							},
						},
					},
				},
				old: []model.Product{
					{
						ID:     "1",
						Handle: "handle-1",
						Variants: &model.ProductVariantConnection{
							Edges: []model.ProductVariantEdge{
								{
									Node: &model.ProductVariant{
										Sku: model.NewString("sku-3"),
									},
								},
							},
						},
					},
				},
			},
			wantToCreate: []model.ProductInput{
				{
					Handle: model.NewString("handle-2"),
					Variants: []model.ProductVariantInput{
						{
							Sku: model.NewString("sku-2"),
						},
					},
				},
			},
			wantToUpdate:         []model.ProductInput{},
			wantToCreateVariants: []variantBulkCreateInput{},
		},
		{
			name: "matches by the variants' full set of sku",
			args: args{
				new: []model.ProductInput{
					{
						Handle: model.NewString("handle-1-new"),
						Variants: []model.ProductVariantInput{
							{
								Sku: model.NewString("sku-1"),
							},
							{
								Sku: model.NewString("sku-2"),
							},
						},
					},
				},
				old: []model.Product{
					{
						ID:     "1",
						Handle: "handle-1",
						Variants: &model.ProductVariantConnection{
							Edges: []model.ProductVariantEdge{
								{
									Node: &model.ProductVariant{
										Sku: model.NewString("sku-1"),
									},
								},
								{
									Node: &model.ProductVariant{
										Sku: model.NewString("sku-2"),
									},
								},
							},
						},
					},
				},
			},
			wantToCreate:         []model.ProductInput{},
			wantToUpdate:         []model.ProductInput{},
			wantToCreateVariants: []variantBulkCreateInput{},
		},
		{
			name: "when for a new handle the new set matches by the variants' set of sku partially, the rest won't be created",
			args: args{
				new: []model.ProductInput{
					{
						Handle: model.NewString("handle-1-new"),
						Variants: []model.ProductVariantInput{
							{
								Sku: model.NewString("sku-1"),
							},
							{
								Sku: model.NewString("sku-2"),
							},
							{
								Sku: model.NewString("sku-3"),
							},
							{
								Sku: model.NewString("sku-4"),
							},
						},
					},
				},
				old: []model.Product{
					{
						ID:     "1",
						Handle: "handle-1",
						Variants: &model.ProductVariantConnection{
							Edges: []model.ProductVariantEdge{
								{
									Node: &model.ProductVariant{
										Sku: model.NewString("sku-1"),
									},
								},
								{
									Node: &model.ProductVariant{
										Sku: model.NewString("sku-2"),
									},
								},
							},
						},
					},
				},
			},
			wantToCreate:         []model.ProductInput{},
			wantToUpdate:         []model.ProductInput{},
			wantToCreateVariants: []variantBulkCreateInput{},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			const overwriteProducts = false
			const addMissingVariants = false
			gotToCreate, gotToUpdate, gotToCreateVariants, err := mergeProductsByHandleAndSKU(tt.args.new, tt.args.old, overwriteProducts, addMissingVariants)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.wantToCreate, gotToCreate)
			assert.Equal(t, tt.wantToUpdate, gotToUpdate)
			assert.Equal(t, tt.wantToCreateVariants, gotToCreateVariants)
		})
	}
}

func Test_dedupProductsByHandle_do_not_overwrite_and_create_missing_variants(t *testing.T) {
	type args struct {
		new []model.ProductInput
		old []model.Product
	}
	tests := []struct {
		name                 string
		args                 args
		wantToCreate         []model.ProductInput
		wantToUpdate         []model.ProductInput
		wantToCreateVariants []variantBulkCreateInput
		wantError            bool
	}{
		{
			name: "no products matching by handle — a product will be created",
			args: args{
				new: []model.ProductInput{
					{
						Handle: model.NewString("handle-1"),
						Variants: []model.ProductVariantInput{
							{
								Sku: model.NewString("sku-1"),
							},
						},
					},
				},
				old: []model.Product{
					{
						ID:     "2",
						Handle: "handle-2",
						Variants: &model.ProductVariantConnection{
							Edges: []model.ProductVariantEdge{
								{
									Node: &model.ProductVariant{
										Sku: model.NewString("sku-2"),
									},
								},
							},
						},
					},
				},
			},
			wantToCreate: []model.ProductInput{
				{
					Handle: model.NewString("handle-1"),
					Variants: []model.ProductVariantInput{
						{
							Sku: model.NewString("sku-1"),
						},
					},
				},
			},
			wantToUpdate:         []model.ProductInput{},
			wantToCreateVariants: []variantBulkCreateInput{},
		},
		{
			name: "both create and create missing variants cases",
			args: args{
				new: []model.ProductInput{
					{
						Handle: model.NewString("handle-2"),
						Variants: []model.ProductVariantInput{
							{
								Sku: model.NewString("sku-2"),
							},
						},
					},
					{
						Handle: model.NewString("handle-1"),
						Variants: []model.ProductVariantInput{
							{
								Sku: model.NewString("sku-1"),
							},
						},
					},
				},
				old: []model.Product{
					{
						ID:     "1",
						Handle: "handle-1",
						Variants: &model.ProductVariantConnection{
							Edges: []model.ProductVariantEdge{
								{
									Node: &model.ProductVariant{
										Sku: model.NewString("sku-3"),
									},
								},
							},
						},
					},
				},
			},
			wantToCreate: []model.ProductInput{
				{
					Handle: model.NewString("handle-2"),
					Variants: []model.ProductVariantInput{
						{
							Sku: model.NewString("sku-2"),
						},
					},
				},
			},
			wantToUpdate: []model.ProductInput{},
			wantToCreateVariants: []variantBulkCreateInput{
				{
					ProductID:     "1",
					ProductHandle: "handle-1",
					ProductVariantsBulkInput: []model.ProductVariantsBulkInput{
						{
							Sku: model.NewString("sku-1"),
						},
					},
				},
			},
		},
		{
			name: "matches by the variants' full set of sku",
			args: args{
				new: []model.ProductInput{
					{
						Handle: model.NewString("handle-1-new"),
						Variants: []model.ProductVariantInput{
							{
								Sku: model.NewString("sku-1"),
							},
							{
								Sku: model.NewString("sku-2"),
							},
						},
					},
				},
				old: []model.Product{
					{
						ID:     "1",
						Handle: "handle-1",
						Variants: &model.ProductVariantConnection{
							Edges: []model.ProductVariantEdge{
								{
									Node: &model.ProductVariant{
										Sku: model.NewString("sku-1"),
									},
								},
								{
									Node: &model.ProductVariant{
										Sku: model.NewString("sku-2"),
									},
								},
							},
						},
					},
				},
			},
			wantToCreate:         []model.ProductInput{},
			wantToUpdate:         []model.ProductInput{},
			wantToCreateVariants: []variantBulkCreateInput{},
		},
		{
			name: "when for a new handle the new set matches by the variants' set of sku partially, the rest will be created",
			args: args{
				new: []model.ProductInput{
					{
						Handle: model.NewString("handle-1-new"),
						Variants: []model.ProductVariantInput{
							{
								Sku: model.NewString("sku-1"),
							},
							{
								Sku: model.NewString("sku-2"),
							},
							{
								Sku: model.NewString("sku-3"),
							},
							{
								Sku: model.NewString("sku-4"),
							},
						},
					},
				},
				old: []model.Product{
					{
						ID:     "1",
						Handle: "handle-1",
						Variants: &model.ProductVariantConnection{
							Edges: []model.ProductVariantEdge{
								{
									Node: &model.ProductVariant{
										Sku: model.NewString("sku-1"),
									},
								},
								{
									Node: &model.ProductVariant{
										Sku: model.NewString("sku-2"),
									},
								},
							},
						},
					},
				},
			},
			wantToCreate: []model.ProductInput{},
			wantToUpdate: []model.ProductInput{},
			wantToCreateVariants: []variantBulkCreateInput{
				{
					ProductID:     "1",
					ProductHandle: "handle-1",
					ProductVariantsBulkInput: []model.ProductVariantsBulkInput{
						{
							Sku: model.NewString("sku-3"),
						},
						{
							Sku: model.NewString("sku-4"),
						},
					},
				},
			},
		},
		{
			name: "when the variant to be created duplicates existing variant, an error returned",
			args: args{
				new: []model.ProductInput{
					{
						// new product with only some of the SKUs matching
						Handle: model.NewString("handle-2-new"),
						Variants: []model.ProductVariantInput{
							{
								Sku: model.NewString("sku-2-1"),
							},
							{
								Sku: model.NewString("sku-2-3"),
							},
						},
					},
				},
				old: []model.Product{
					{
						ID:     "1",
						Handle: "handle-1",
						Variants: &model.ProductVariantConnection{
							Edges: []model.ProductVariantEdge{
								{
									Node: &model.ProductVariant{
										Sku: model.NewString("sku-2-3"),
									},
								},
							},
						},
					},
					{
						ID:     "2",
						Handle: "handle-2",
						Variants: &model.ProductVariantConnection{
							Edges: []model.ProductVariantEdge{
								{
									Node: &model.ProductVariant{
										Sku: model.NewString("sku-2-1"),
									},
								},
								{
									Node: &model.ProductVariant{
										Sku: model.NewString("sku-2-2"),
									},
								},
							},
						},
					},
				},
			},
			wantError: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			const overwriteProducts = false
			const addMissingVariants = true
			gotToCreate, gotToUpdate, gotToCreateVariants, err := mergeProductsByHandleAndSKU(tt.args.new, tt.args.old, overwriteProducts, addMissingVariants)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.wantToCreate, gotToCreate)
			assert.Equal(t, tt.wantToUpdate, gotToUpdate)
			assert.Equal(t, tt.wantToCreateVariants, gotToCreateVariants)
		})
	}
}

// {
// 	name: "SKUs the same — the product will be updated",
// 	args: args{
// 		new: []model.ProductInput{
// 			{
// 				Variants: []model.ProductVariantInput{
// 					{
// 						Sku: model.NewString("sku-1"),
// 					},
// 					{
// 						Sku: model.NewString("sku-2"),
// 					},
// 				},
// 				Metafields: []*model.MetafieldInput{
// 					{
// 						Namespace: model.NewString("meta-1"),
// 						Key:       model.NewString("key-1"),
// 						Value:     model.NewString("val-1"),
// 					},
// 					{
// 						Namespace: model.NewString("meta-2"),
// 						Key:       model.NewString("key-2"),
// 						Value:     model.NewString("val-2"),
// 					},
// 				},
// 			},
// 		},
// 		old: []model.Product{
// 			{
// 				ID: "1"),
// 				Variants: &model.ProductVariantConnection{
// 					Edges: []model.ProductVariantEdge{
// 						{
// 							Node: &model.ProductVariant{
// 								Sku: model.NewString("sku-1"),
// 							},
// 						},
// 						{
// 							Node: &model.ProductVariant{
// 								Sku: model.NewString("sku-2"),
// 							},
// 						},
// 					},
// 				},
// 				Metafields: &model.MetafieldConnection{
// 					Edges: []*model.MetafieldEdge{
// 						{
// 							Node: &model.Metafield{
// 								ID:        "metafield-2"),
// 								Namespace: "meta-2"),
// 								Key:       "key-2"),
// 								Value:     "val-2"),
// 							},
// 						},
// 					},
// 				},
// 			},
// 		},
// 		overwrite: true,
// 	},
// 	wantToCreate: []model.ProductInput{},
// 	wantToUpdate: []model.ProductInput{
// 		{
// 			ID: model.NewString("1"),
// 			Variants: []model.ProductVariantInput{{
// 				Sku: model.NewString("sku-1"),
// 			}, {
// 				Sku: model.NewString("sku-2"),
// 			}},
// 			Metafields: []*model.MetafieldInput{{
// 				Namespace: model.NewString("meta-1"),
// 				Key:       model.NewString("key-1"),
// 				Value:     model.NewString("val-1"),
// 			}, {
// 				ID:        model.NewString("metafield-2"),
// 				Namespace: model.NewString("meta-2"),
// 				Key:       model.NewString("key-2"),
// 				Value:     model.NewString("val-2"),
// 			}},
// 		},
// 	},
// },
// {
// 	name: "SKUs match partially — the product will be updated",
// 	args: args{
// 		new: []model.ProductInput{
// 			{
// 				Variants: []model.ProductVariantInput{
// 					{
// 						Sku: model.NewString("sku-3"),
// 					},
// 					{
// 						Sku: model.NewString("sku-2"),
// 					},
// 				},
// 			},
// 		},
// 		old: []model.Product{
// 			{
// 				ID: "1"),
// 				Variants: &model.ProductVariantConnection{
// 					Edges: []model.ProductVariantEdge{
// 						{
// 							Node: &model.ProductVariant{
// 								Sku: model.NewString("sku-1"),
// 							},
// 						},
// 						{
// 							Node: &model.ProductVariant{
// 								Sku: model.NewString("sku-2"),
// 							},
// 						},
// 					},
// 				},
// 			},
// 		},
// 		overwrite: true,
// 	},
// 	wantToCreate: []model.ProductInput{},
// 	wantToUpdate: []model.ProductInput{
// 		{
// 			ID: model.NewString("1"),
// 			Variants: []model.ProductVariantInput{{
// 				Sku: model.NewString("sku-3"),
// 			}, {
// 				Sku: model.NewString("sku-2"),
// 			}},
// 		},
// 	},
// },
// {
// 	name: "SKUs match and overwrite is false — the product will be skipped",
// 	args: args{
// 		new: []model.ProductInput{
// 			{
// 				Variants: []model.ProductVariantInput{
// 					{
// 						Sku: model.NewString("sku-3"),
// 					},
// 					{
// 						Sku: model.NewString("sku-2"),
// 					},
// 				},
// 			},
// 		},
// 		old: []model.Product{
// 			{
// 				ID: "1"),
// 				Variants: &model.ProductVariantConnection{
// 					Edges: []model.ProductVariantEdge{
// 						{
// 							Node: &model.ProductVariant{
// 								Sku: model.NewString("sku-1"),
// 							},
// 						},
// 						{
// 							Node: &model.ProductVariant{
// 								Sku: model.NewString("sku-2"),
// 							},
// 						},
// 					},
// 				},
// 			},
// 		},
// 		overwrite: false,
// 	},
// 	wantToCreate: []model.ProductInput{},
// 	wantToUpdate: []model.ProductInput{},
// },
