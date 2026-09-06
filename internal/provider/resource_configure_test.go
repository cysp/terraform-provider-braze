package provider_test

import (
	"context"
	"testing"

	. "github.com/cysp/terraform-provider-braze/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type configurableResource interface {
	Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse)
}

func TestResourceConfigureDiagnostics(t *testing.T) {
	t.Parallel()

	configuredProvider := NewBrazeProvider("test")

	resources := make([]configurableResource, 0, 8)

	for _, factory := range configuredProvider.Resources(t.Context()) {
		configured, ok := factory().(configurableResource)
		require.True(t, ok)

		resources = append(resources, configured)
	}

	for _, factory := range configuredProvider.ListResources(t.Context()) {
		configured, ok := factory().(configurableResource)
		require.True(t, ok)

		resources = append(resources, configured)
	}

	for _, r := range resources {
		var response resource.ConfigureResponse
		r.Configure(t.Context(), resource.ConfigureRequest{}, &response)
		assert.False(t, response.Diagnostics.HasError(), "%T must tolerate validation before provider configuration", r)

		r.Configure(t.Context(), resource.ConfigureRequest{ProviderData: "wrong type"}, &response)
		assert.True(t, response.Diagnostics.HasError(), "%T must report invalid provider data", r)
	}
}
