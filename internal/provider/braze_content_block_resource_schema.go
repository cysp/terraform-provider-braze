package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func BrazeContentBlockResourceIdentitySchema() identityschema.Schema {
	return identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{
				RequiredForImport: true,
			},
		},
	}
}

func BrazeContentBlockResourceSchema(ctx context.Context) (schema.Schema, diag.Diagnostics) {
	diags := diag.Diagnostics{}

	emptyTypedListOfString := NewTypedList([]types.String{})

	emptyTypedListOfStringListValue, emptyTypedListOfStringListValueDiags := emptyTypedListOfString.ToListValue(ctx)
	diags.Append(emptyTypedListOfStringListValueDiags...)

	schema := schema.Schema{
		Description: "Manage Braze Content Blocks.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"content": schema.StringAttribute{
				Description: "Content",
				Required:    true,
			},
			"tags": schema.ListAttribute{
				Description: "Tags",
				CustomType:  NewTypedListNull[types.String]().CustomType(ctx),
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     listdefault.StaticValue(emptyTypedListOfStringListValue),
			},
		},
	}

	return schema, diags
}
