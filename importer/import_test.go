package importer

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/r0busta/go-shopify-graphql/v2"
)

func Test_dedupProductsBySKU(t *testing.T) {
	type args struct {
		new       []*shopify.ProductCreate
		old       []*shopify.ProductBulkResult
		overwrite bool
	}
	tests := []struct {
		name         string
		args         args
		wantToCreate []*shopify.ProductCreate
		wantToUpdate []*shopify.ProductUpdate
	}{
		{
			name: "SKUs don't match– the product will be created",
			args: args{
				new: []*shopify.ProductCreate{{
					ProductInput: shopify.ProductInput{
						Variants: []shopify.ProductVariantInput{{
							SKU: "sku-3",
						}},
					},
				}},
				old: []*shopify.ProductBulkResult{{
					ProductBase: shopify.ProductBase{
						ID: "1",
					},
					ProductVariants: []shopify.ProductVariant{{
						SKU: "sku-1",
					}, {
						SKU: "sku-2",
					}},
				}},
				overwrite: true,
			},
			wantToCreate: []*shopify.ProductCreate{{
				ProductInput: shopify.ProductInput{
					Variants: []shopify.ProductVariantInput{{
						SKU: "sku-3",
					}},
				},
			}},
			wantToUpdate: []*shopify.ProductUpdate{},
		},
		{
			name: "SKUs the same– the product will be updated",
			args: args{
				new: []*shopify.ProductCreate{{
					ProductInput: shopify.ProductInput{
						Variants: []shopify.ProductVariantInput{{
							SKU: "sku-1",
						}, {
							SKU: "sku-2",
						}},
						Metafields: []shopify.MetafieldInput{{
							Namespace: "meta-1",
							Key:       "key-1",
							Value:     "val-1",
						}, {
							Namespace: "meta-2",
							Key:       "key-2",
							Value:     "val-2",
						}},
					},
				}},
				old: []*shopify.ProductBulkResult{{
					ProductBase: shopify.ProductBase{
						ID: "1",
					},
					ProductVariants: []shopify.ProductVariant{{
						SKU: "sku-1",
					}, {
						SKU: "sku-2",
					}},
					Metafields: []shopify.Metafield{{
						ID:        "metafield-2",
						Namespace: "meta-2",
						Key:       "key-2",
						Value:     "val-2",
					}},
				}},
				overwrite: true,
			},
			wantToCreate: []*shopify.ProductCreate{},
			wantToUpdate: []*shopify.ProductUpdate{{
				ProductInput: shopify.ProductInput{
					ID: "1",
					Variants: []shopify.ProductVariantInput{{
						SKU: "sku-1",
					}, {
						SKU: "sku-2",
					}},
					Metafields: []shopify.MetafieldInput{{
						Namespace: "meta-1",
						Key:       "key-1",
						Value:     "val-1",
					}, {
						ID:        "metafield-2",
						Namespace: "meta-2",
						Key:       "key-2",
						Value:     "val-2",
					}},
				},
			}},
		},
		{
			name: "SKUs match partially– the product will be updated",
			args: args{
				new: []*shopify.ProductCreate{{
					ProductInput: shopify.ProductInput{
						Variants: []shopify.ProductVariantInput{{
							SKU: "sku-3",
						}, {
							SKU: "sku-2",
						}},
					},
				}},
				old: []*shopify.ProductBulkResult{{
					ProductBase: shopify.ProductBase{
						ID: "1",
					},
					ProductVariants: []shopify.ProductVariant{{
						SKU: "sku-1",
					}, {
						SKU: "sku-2",
					}},
				}},
				overwrite: true,
			},
			wantToCreate: []*shopify.ProductCreate{},
			wantToUpdate: []*shopify.ProductUpdate{{
				ProductInput: shopify.ProductInput{
					ID: "1",
					Variants: []shopify.ProductVariantInput{{
						SKU: "sku-3",
					}, {
						SKU: "sku-2",
					}},
				},
			}},
		},
		{
			name: "SKUs match and overwrite is false– the product will be skipped",
			args: args{
				new: []*shopify.ProductCreate{{
					ProductInput: shopify.ProductInput{
						Variants: []shopify.ProductVariantInput{{
							SKU: "sku-3",
						}, {
							SKU: "sku-2",
						}},
					},
				}},
				old: []*shopify.ProductBulkResult{{
					ProductBase: shopify.ProductBase{
						ID: "1",
					},
					ProductVariants: []shopify.ProductVariant{{
						SKU: "sku-1",
					}, {
						SKU: "sku-2",
					}},
				}},
				overwrite: false,
			},
			wantToCreate: []*shopify.ProductCreate{},
			wantToUpdate: []*shopify.ProductUpdate{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotToCreate, gotToUpdate := dedupProductsBySKU(tt.args.new, tt.args.old, tt.args.overwrite)
			if !reflect.DeepEqual(gotToCreate, tt.wantToCreate) {
				t.Errorf("dedupProductsBySKU() gotToCreate = %v, want %v", gotToCreate, tt.wantToCreate)
				d, _ := json.MarshalIndent(gotToCreate, "", "\t")
				fmt.Println(string(d))
			}
			if !reflect.DeepEqual(gotToUpdate, tt.wantToUpdate) {
				t.Errorf("dedupProductsBySKU() gotToUpdate = %v, want %v", gotToUpdate, tt.wantToUpdate)
				d, _ := json.MarshalIndent(gotToUpdate, "", "\t")
				fmt.Println(string(d))
			}
		})
	}
}
