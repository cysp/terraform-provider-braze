package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
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

func BrazeEmailTemplateResourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Manage Braze Email Templates stored on the Templates & Media page.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"template_name": schema.StringAttribute{
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
				Description: "A list of tags to categorize the email template.",
				CustomType:  NewTypedListNull[types.String]().CustomType(ctx),
				ElementType: types.StringType,
				Optional:    true,
			},
			"should_inline_css": schema.BoolAttribute{
				Description: "Whether Braze should inline CSS for this template. When unset, Braze uses the App Group default.",
				Optional:    true,
			},
		},
	}
}
