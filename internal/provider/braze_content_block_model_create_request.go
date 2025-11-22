package provider

import (
	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
)

func (m brazeContentBlockModel) ToCreateContentBlockRequest() brazeclient.CreateContentBlockRequest {
	req := brazeclient.CreateContentBlockRequest{
		Name:        m.Name.ValueString(),
		Description: brazeclient.NewOptPointerString(m.Description.ValueStringPointer()),
		Content:     m.Content.ValueString(),
		Tags:        TypedListToStringSlice(m.Tags),
	}

	return req
}
