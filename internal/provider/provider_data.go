package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func setProviderData(data any, out *brazeProviderData) diag.Diagnostics {
	if data == nil {
		return nil
	}

	if providerData, ok := data.(brazeProviderData); ok {
		*out = providerData

		return nil
	}

	return diag.Diagnostics{diag.NewErrorDiagnostic(
		"Invalid provider data",
		fmt.Sprintf("Expected the configured Braze client, got %T. Please report this provider error.", data),
	)}
}
