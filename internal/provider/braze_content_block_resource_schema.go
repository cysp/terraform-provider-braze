package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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

func BrazeContentBlockResourceSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		Description: "Manage Braze Content Blocks, reusable snippets for messaging campaigns.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:        "The Braze-generated ID. Use import to manage an existing object.",
				DeprecationMessage: "Configuring id is deprecated. Remove it from the resource configuration; Terraform retains the existing ID in state. Use import to adopt existing objects.",
				Optional:           true,
				Computed:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Validators:  []validator.String{nonBlankStringValidator{}},
				Description: "A unique name using letters, numbers, hyphens, and underscores. Braze does not permit renaming active content blocks.",
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
				Description: "Tags that already exist in Braze. Null elements are invalid; an empty list is sent as an empty array.",
				Validators:  []validator.List{tagsValidator{}},
				ElementType: types.StringType,
				Optional:    true,
			},
		},
	}
}
