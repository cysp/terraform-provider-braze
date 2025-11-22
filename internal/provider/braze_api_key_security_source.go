package provider

import (
	"context"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
)

type BrazeAPIKeySecuritySource struct {
	token string
}

var _ brazeclient.SecuritySource = (*BrazeAPIKeySecuritySource)(nil)

func NewBrazeAPIKeySecuritySource(token string) BrazeAPIKeySecuritySource {
	return BrazeAPIKeySecuritySource{
		token: token,
	}
}

//revive:disable:var-naming
func (source BrazeAPIKeySecuritySource) BrazeApiKey(_ context.Context, _ brazeclient.OperationName, _ *brazeclient.Client) (brazeclient.BrazeApiKey, error) {
	return brazeclient.BrazeApiKey{Token: source.token}, nil
}
