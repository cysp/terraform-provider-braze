package provider

import (
	"context"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
)

func (m brazeContentBlockModel) ToUpdateContentBlockRequest(ctx context.Context) (brazeclient.UpdateContentBlockRequest, error) {
	req := brazeclient.UpdateContentBlockRequest{
		ContentBlockID: m.ID.ValueString(),
		Name:           brazeclient.NewOptString(m.Name.ValueString()),
		Description:    brazeclient.NewOptNilPointerString(m.Description.ValueStringPointer()),
		Content:        brazeclient.NewOptString(m.Content.ValueString()),
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
