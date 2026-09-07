package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func BrazeSDKAuthenticationKeyResourceIdentitySchema() identityschema.Schema {
	return identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"app_id": identityschema.StringAttribute{RequiredForImport: true},
			"id":     identityschema.StringAttribute{RequiredForImport: true},
		},
	}
}

func BrazeSDKAuthenticationKeyResourceSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		Description: "Manage one RSA public key for Braze SDK Authentication in one app. Keys outside this resource remain unmanaged. The current primary key cannot be deleted; rotate with separate apply stages before retiring it.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The SDK Authentication key ID assigned by Braze.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"app_id": schema.StringAttribute{
				Description: "The Braze app API identifier.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"rsa_public_key": schema.StringAttribute{
				Description: "The RSA public key in PEM format. Braze recommends a 2048-bit RSA key for RS256 JWT signatures.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "A non-empty description of the key. Braze does not provide an endpoint to update it, so changes replace the key.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"primary": schema.BoolAttribute{
				Description: "The primary-role claim for this key. Set to true to make this key primary when created and to re-promote it after drift. Omit it to leave primary selection unmanaged; state still reports whether Braze currently considers the key primary. Configure at most one primary key per app. False is unsupported because Braze can only demote a key by promoting another key.",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}
