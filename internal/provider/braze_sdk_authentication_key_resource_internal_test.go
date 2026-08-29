package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
)

func TestBrazeSDKAuthenticationKeyImportRejectsWhitespaceOnlyIdentifiers(t *testing.T) {
	t.Parallel()

	t.Run("composite identifier", func(t *testing.T) {
		t.Parallel()

		req := resource.ImportStateRequest{ID: " /key-id"}
		resp := resource.ImportStateResponse{}

		(&brazeSDKAuthenticationKeyResource{}).ImportState(t.Context(), req, &resp)

		assert.True(t, resp.Diagnostics.HasError())
	})

	t.Run("resource identity", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		identitySchema := BrazeSDKAuthenticationKeyResourceIdentitySchema()
		identityType := identitySchema.Type().TerraformType(ctx)
		identity := &tfsdk.ResourceIdentity{
			Schema: identitySchema,
			Raw: tftypes.NewValue(identityType, map[string]tftypes.Value{
				"app_id": tftypes.NewValue(tftypes.String, " "),
				"id":     tftypes.NewValue(tftypes.String, "key-id"),
			}),
		}
		req := resource.ImportStateRequest{Identity: identity}
		resp := resource.ImportStateResponse{Identity: identity}

		(&brazeSDKAuthenticationKeyResource{}).ImportState(ctx, req, &resp)

		assert.True(t, resp.Diagnostics.HasError())
	})
}
