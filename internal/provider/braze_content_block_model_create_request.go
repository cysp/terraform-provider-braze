package provider

import (
	"context"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
)

func (m brazeContentBlockModel) ToCreateContentBlockRequest(ctx context.Context) (brazeclient.CreateContentBlockRequest, error) {
	req := brazeclient.CreateContentBlockRequest{
		Name:        m.Name.ValueString(),
		Description: brazeclient.NewOptNilPointerString(m.Description.ValueStringPointer()),
		Content:     m.Content.ValueString(),
	}

	tags, err := stringListToSlice(ctx, m.Tags)
	if err != nil {
		return req, err
	}

	if tags != nil {
		req.Tags.SetTo(tags)
	} else {
		req.Tags.SetToNull()
	}

	return req, nil
}
