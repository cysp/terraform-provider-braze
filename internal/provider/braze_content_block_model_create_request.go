package provider

import (
	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
)

func (m brazeContentBlockModel) ToCreateContentBlockRequest() brazeclient.CreateContentBlockRequest {
	req := brazeclient.CreateContentBlockRequest{
		Name:    m.Name.ValueString(),
		Content: m.Content.ValueString(),
		Tags:    TypedListToStringSlice(m.Tags),
	}

	description := m.Description.ValueStringPointer()
	if description != nil {
		req.Description.SetTo(*description)
	} else {
		req.Description.SetToNull()
	}

	return req
}
