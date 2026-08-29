package provider

import (
	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type brazeSDKAuthenticationKeyModel struct {
	ID           types.String `tfsdk:"id"`
	AppID        types.String `tfsdk:"app_id"`
	RSAPublicKey types.String `tfsdk:"rsa_public_key"`
	Description  types.String `tfsdk:"description"`
	Primary      types.Bool   `tfsdk:"primary"`
}

func newBrazeSDKAuthenticationKeyModel(
	appID string,
	key brazeclient.SDKAuthenticationKey,
) brazeSDKAuthenticationKeyModel {
	return brazeSDKAuthenticationKeyModel{
		ID:           types.StringValue(key.GetID()),
		AppID:        types.StringValue(appID),
		RSAPublicKey: types.StringValue(key.GetRsaPublicKey()),
		Description:  types.StringValue(key.GetDescription()),
		Primary:      types.BoolValue(key.GetIsPrimary()),
	}
}
