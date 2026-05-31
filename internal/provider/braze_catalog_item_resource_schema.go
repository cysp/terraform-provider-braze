package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func BrazeCatalogItemResourceIdentitySchema() identityschema.Schema {
	return identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{RequiredForImport: true},
		},
	}
}

func BrazeCatalogItemResourceSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		Description: "Manage a Braze catalog item using canonical JSON for arbitrary catalog item values.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The Terraform import ID in `catalog_name/item_id` form.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"catalog_name": schema.StringAttribute{
				Description: "The name of the catalog containing the item.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"item_id": schema.StringAttribute{
				Description: "The catalog item ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"values_json": schema.StringAttribute{
				CustomType:  jsontypes.NormalizedType{},
				Description: "Canonical JSON object containing the item values sent in the request body. The Braze `id` field is addressed by `item_id` and must not be included.",
				Required:    true,
			},
		},
	}
}
