package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type brazeContentBlockModel struct {
	IDIdentityModel

	Name        types.String            `tfsdk:"name"`
	Description types.String            `tfsdk:"description"`
	Content     types.String            `tfsdk:"content"`
	Tags        TypedList[types.String] `tfsdk:"tags"`
}
