package provider

import (
	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
)

func (m brazeContentBlockModel) ToCreateContentBlockRequest() brazeclient.CreateContentBlockRequest {
	req := brazeclient.CreateContentBlockRequest{
		Name:        m.Name.ValueString(),
		Description: brazeclient.NewOptNilPointerString(m.Description.ValueStringPointer()),
		Content:     m.Content.ValueString(),
	}

	tags := TypedListToStringSlice(m.Tags)
	if tags != nil {
		req.Tags.SetTo(tags)
	} else {
		req.Tags.SetToNull()
	}

	return req
}
