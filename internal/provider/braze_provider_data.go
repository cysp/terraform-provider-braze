package provider

import (
	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
)

type brazeProviderData struct {
	client *brazeclient.Client
}
