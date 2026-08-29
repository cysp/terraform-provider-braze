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
		Description: "Manage one RSA public key in the bounded set used by Braze SDK Authentication for an app. The provider REST API key needs the `sdk_authentication.create`, `sdk_authentication.keys`, `sdk_authentication.primary`, and `sdk_authentication.delete` permissions for the full lifecycle. This resource registers public-key material only; it does not manage the private key or enable SDK Authentication enforcement.\n\nBraze deletion rule: Braze rejects deletion of whichever key it currently reports as primary, even when the app has other keys. The provider reports this during planning. Promote another key and apply that change first; after refresh reports the old key as non-primary, remove it in a later apply. A destroy of a managed set containing the currently primary key therefore cannot complete through the Braze API. Use a `removed` block with `destroy = false` only when intentionally relinquishing management without deleting the key.\n\nProvider replacement policy: the provider rejects replacement of a currently primary key. Terraform's default destroy-before-create order is guaranteed to fail, while `create_before_destroy` compresses creation, promotion, and deletion into one apply and cannot represent a controlled JWT issuer-migration interval. Rotate with stable resource addresses: add and promote the replacement while retaining the old key, migrate JWT issuers, then remove the old key in a later apply.\n\nConfigure `primary = true` on at most one key per app and use `null` or omission on every other key.",
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
