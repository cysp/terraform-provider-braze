package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func BrazeCatalogFieldObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"name": types.StringType,
		"type": types.StringType,
	}}
}

func BrazeCatalogResourceIdentitySchema() identityschema.Schema {
	return identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"name": identityschema.StringAttribute{RequiredForImport: true},
		},
	}
}

func BrazeCatalogResourceSchema(ctx context.Context) schema.Schema {
	_ = ctx

	return schema.Schema{
		Description: "Manage Braze catalogs and their creation-time field schema.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The catalog name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "The catalog description.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"fields": schema.ListNestedAttribute{
				Description: "The catalog field schema. Braze requires the first field to be `id` with type `string`.",
				Required:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{Required: true},
						"type": schema.StringAttribute{Required: true},
					},
				},
			},
			"num_items": schema.Int64Attribute{
				Description: "The number of items in the catalog.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The time the catalog was last updated.",
				Computed:    true,
			},
		},
	}
}
