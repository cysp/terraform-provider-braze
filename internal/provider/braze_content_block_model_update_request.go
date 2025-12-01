package provider

import (
	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
)

func (m brazeContentBlockModel) ToUpdateContentBlockRequest() brazeclient.UpdateContentBlockRequest {
	req := brazeclient.UpdateContentBlockRequest{
		ContentBlockID: m.ID.ValueString(),
		Name:           brazeclient.NewOptString(m.Name.ValueString()),
		Description:    brazeclient.NewOptNilPointerString(m.Description.ValueStringPointer()),
		Content:        brazeclient.NewOptString(m.Content.ValueString()),
	}

	tags := TypedListToStringSlice(m.Tags)
	if tags != nil {
		req.Tags.SetTo(tags)
	} else {
		req.Tags.SetToNull()
	}

	return req
}
