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

func BrazeEmailTemplateResourceIdentitySchema() identityschema.Schema {
	return identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{
				RequiredForImport: true,
			},
		},
	}
}

func BrazeEmailTemplateResourceSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		Description: "Manage Braze Email Templates stored on the Templates & Media page. Templates created with the drag-and-drop editor are not supported.",
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
			"template_name": schema.StringAttribute{
				Validators:  []validator.String{nonBlankStringValidator{}},
				Description: "The name of the email template.",
				Required:    true,
			},
			"subject": schema.StringAttribute{
				Description: "The email template subject line.",
				Required:    true,
			},
			"body": schema.StringAttribute{
				Description: "The email template body, which may include HTML.",
				Required:    true,
			},
			"plaintext_body": schema.StringAttribute{
				Description: "A plaintext version of the email template body.",
				Optional:    true,
			},
			"preheader": schema.StringAttribute{
				Description: "The email preheader used to generate previews in some clients.",
				Optional:    true,
			},
			"tags": schema.ListAttribute{
				Description: "Tags that already exist in Braze. Null elements are invalid; an empty list is sent as an empty array.",
				Validators:  []validator.List{tagsValidator{}},
				ElementType: types.StringType,
				Optional:    true,
			},
			"should_inline_css": schema.BoolAttribute{
				Description: "Whether Braze should inline CSS for this template. Omit on creation to use the App Group default. When omitted for an existing template, its observed value is retained. Set explicitly to manage this setting.",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}
