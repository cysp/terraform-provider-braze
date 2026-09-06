package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type brazeSDKAuthenticationKeysModel struct {
	AppID types.String `tfsdk:"app_id"`
	Keys  types.Set    `tfsdk:"keys"`
}

type brazeSDKAuthenticationKeysMemberModel struct {
	ID           types.String `tfsdk:"id"`
	RSAPublicKey types.String `tfsdk:"rsa_public_key"`
	Description  types.String `tfsdk:"description"`
	Primary      types.Bool   `tfsdk:"primary"`
}

func sdkAuthenticationKeysMemberType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"id":             types.StringType,
		"rsa_public_key": types.StringType,
		"description":    types.StringType,
		"primary":        types.BoolType,
	}}
}

// IDs are computed and do not participate in matching configured members.
func (m brazeSDKAuthenticationKeysModel) collection(
	ctx context.Context,
	includeIDs bool,
) ([]sdkAuthenticationCollectionKey, bool, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	if m.Keys.IsNull() || m.Keys.IsUnknown() {
		return nil, false, diags
	}

	keys := make([]sdkAuthenticationCollectionKey, 0, len(m.Keys.Elements()))
	for _, element := range m.Keys.Elements() {
		if element.IsNull() || element.IsUnknown() {
			return nil, false, diags
		}

		var member brazeSDKAuthenticationKeysMemberModel

		object, ok := element.(types.Object)
		if !ok {
			diags.AddError("Invalid SDK Authentication key state", "Expected an object in the keys set.")

			return nil, false, diags
		}

		diags.Append(object.As(ctx, &member, basetypes.ObjectAsOptions{})...)

		if diags.HasError() {
			return nil, false, diags
		}

		if member.RSAPublicKey.IsUnknown() || member.Description.IsUnknown() || member.Primary.IsUnknown() ||
			member.RSAPublicKey.IsNull() || member.Description.IsNull() || member.Primary.IsNull() ||
			(includeIDs && (member.ID.IsUnknown() || member.ID.IsNull())) {
			return nil, false, diags
		}

		key := sdkAuthenticationCollectionKey{
			RSAPublicKey: member.RSAPublicKey.ValueString(),
			Description:  member.Description.ValueString(),
			Primary:      member.Primary.ValueBool(),
		}
		if includeIDs {
			key.ID = member.ID.ValueString()
		}

		keys = append(keys, key)
	}

	return keys, true, diags
}

func newBrazeSDKAuthenticationKeysModel(
	ctx context.Context,
	appID string,
	keys []sdkAuthenticationCollectionKey,
	primaryVerified bool,
) (brazeSDKAuthenticationKeysModel, diag.Diagnostics) {
	members := make([]brazeSDKAuthenticationKeysMemberModel, 0, len(keys))
	for _, key := range keys {
		primary := types.BoolNull()
		if primaryVerified {
			primary = types.BoolValue(key.Primary)
		}

		members = append(members, brazeSDKAuthenticationKeysMemberModel{
			ID:           types.StringValue(key.ID),
			RSAPublicKey: types.StringValue(key.RSAPublicKey),
			Description:  types.StringValue(key.Description),
			Primary:      primary,
		})
	}

	values, diags := types.SetValueFrom(ctx, sdkAuthenticationKeysMemberType(), members)

	return brazeSDKAuthenticationKeysModel{AppID: types.StringValue(appID), Keys: values}, diags
}
