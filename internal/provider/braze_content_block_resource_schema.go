package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
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
		Description: "Manage Braze Content Blocks, reusable snippets for messaging campaigns.",
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
				Description: "A unique name for the content block.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "An optional description of the content block.",
				Optional:    true,
			},
			"content": schema.StringAttribute{
				Description: "The content of the content block.",
				Required:    true,
			},
			"tags": schema.ListAttribute{
				Description: "A list of tags to categorize the content block.",
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
