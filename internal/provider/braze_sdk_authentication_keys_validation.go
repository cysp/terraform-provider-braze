package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func validateSDKAuthenticationKeysConfig(ctx context.Context, config brazeSDKAuthenticationKeysModel) diag.Diagnostics {
	diags := diag.Diagnostics{}
	if !config.AppID.IsNull() && !config.AppID.IsUnknown() && strings.TrimSpace(config.AppID.ValueString()) == "" {
		diags.AddAttributeError(path.Root("app_id"), "Empty required value", "The app_id value must not be empty.")
	}

	if config.Keys.IsNull() || config.Keys.IsUnknown() {
		return diags
	}

	if len(config.Keys.Elements()) < 1 || len(config.Keys.Elements()) > 3 {
		diags.AddAttributeError(path.Root("keys"), "Invalid SDK Authentication key count", "Configure between one and three SDK Authentication keys.")
	}

	primaryCount := 0
	primaryKnown := true
	materials := make(map[[2]string]bool)

	for _, element := range config.Keys.Elements() {
		if element.IsUnknown() {
			primaryKnown = false

			continue
		}

		if element.IsNull() {
			diags.AddAttributeError(path.Root("keys"), "Invalid SDK Authentication key", "A key member must not be null.")

			continue
		}

		object, ok := element.(types.Object)
		if !ok {
			diags.AddAttributeError(path.Root("keys"), "Invalid SDK Authentication key", "Expected an object in the keys set.")

			continue
		}

		var member brazeSDKAuthenticationKeysMemberModel

		memberDiags := object.As(ctx, &member, basetypes.ObjectAsOptions{})
		diags.Append(memberDiags...)

		if memberDiags.HasError() {
			continue
		}

		diags.Append(validateSDKAuthenticationKeysMember(member, materials)...)

		if member.Primary.IsUnknown() || member.Primary.IsNull() {
			primaryKnown = false
		} else if member.Primary.ValueBool() {
			primaryCount++
		}
	}

	if primaryCount > 1 || (primaryKnown && primaryCount != 1) {
		diags.AddAttributeError(path.Root("keys"), "Invalid primary selection", "Exactly one configured SDK Authentication key must have primary = true.")
	}

	return diags
}

func validateSDKAuthenticationKeysMember(member brazeSDKAuthenticationKeysMemberModel, materials map[[2]string]bool) diag.Diagnostics {
	diags := diag.Diagnostics{}

	for name, value := range map[string]types.String{"rsa_public_key": member.RSAPublicKey, "description": member.Description} {
		if !value.IsUnknown() && !value.IsNull() && strings.TrimSpace(value.ValueString()) == "" {
			diags.AddAttributeError(path.Root("keys"), "Empty required value", "Each key's "+name+" value must not be empty.")
		}
	}

	if !member.RSAPublicKey.IsUnknown() && !member.RSAPublicKey.IsNull() && !member.Description.IsUnknown() && !member.Description.IsNull() {
		material := [2]string{member.RSAPublicKey.ValueString(), member.Description.ValueString()}
		if materials[material] {
			diags.AddAttributeError(path.Root("keys"), "Ambiguous SDK Authentication keys", "Duplicate public-key/description pairs cannot identify separate keys. Configure a distinct pair for each member.")
		}

		materials[material] = true
	}

	return diags
}
