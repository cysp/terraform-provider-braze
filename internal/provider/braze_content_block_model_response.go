package provider

import (
	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

//revive:disable:unexported-return
func NewBrazeContentBlockModelFromGetContentBlockInfoResponse(response brazeclient.GetContentBlockInfoResponse) brazeContentBlockModel {
	model := brazeContentBlockModel{
		IDIdentityModel: IDIdentityModel{ID: types.StringValue(response.GetContentBlockID())},

		Name:        types.StringValue(response.GetName()),
		Description: types.StringPointerValue(response.GetDescription().GetPointer()),
		Content:     types.StringValue(response.GetContent()),
		Tags:        NewTypedListFromStringSlice(response.GetTags()),
	}

	return model
}
