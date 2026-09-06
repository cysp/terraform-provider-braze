package provider

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestSDKAuthenticationKeyCreateReportsObservedPrimary(t *testing.T) {
	t.Parallel()

	r := NewBrazeSDKAuthenticationKeyResource()
	configureHTTPResource(t, r, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if req.Method == http.MethodPost {
			_, _ = w.Write([]byte(`{"id":"created"}`))

			return
		}

		_, _ = w.Write([]byte(`{"keys":[{"id":"created","rsa_public_key":"public-key","description":"Key","is_primary":false}]}`))
	}))
	plan := lifecyclePlan(t, r, brazeSDKAuthenticationKeyModel{ID: types.StringUnknown(), AppID: types.StringValue("app-1"), RSAPublicKey: types.StringValue("public-key"), Description: types.StringValue("Key"), Primary: types.BoolValue(true)})
	response := resource.CreateResponse{State: tfsdk.State{Schema: plan.Schema}, Identity: lifecycleIdentity(t, r)}
	r.Create(t.Context(), resource.CreateRequest{Plan: plan}, &response)
	require.True(t, response.Diagnostics.HasError())

	var observed brazeSDKAuthenticationKeyModel
	require.False(t, response.State.Get(t.Context(), &observed).HasError())
	assert.Equal(t, "created", observed.ID.ValueString())

	var appID, keyID types.String
	require.False(t, response.Identity.GetAttribute(t.Context(), path.Root("app_id"), &appID).HasError())
	require.False(t, response.Identity.GetAttribute(t.Context(), path.Root("id"), &keyID).HasError())
	assert.Equal(t, "app-1", appID.ValueString())
	assert.Equal(t, "created", keyID.ValueString())
	assert.False(t, observed.Primary.ValueBool(), "A failed primary claim must preserve the observed role")
}

func TestSDKAuthenticationKeyCreateRetrySafety(t *testing.T) {
	t.Parallel()

	for name, status := range map[string]int{"ambiguous create": http.StatusServiceUnavailable, "throttled create": http.StatusTooManyRequests} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var creates atomic.Int32

			r := NewBrazeSDKAuthenticationKeyResource()
			configureHTTPResource(t, r, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Content-Type", "application/json")

				if req.Method == http.MethodPost {
					if creates.Add(1) == 1 {
						w.Header().Set("Retry-After", "0")
						w.WriteHeader(status)
						_, _ = w.Write([]byte(`{"message":"retry later"}`))

						return
					}

					_, _ = w.Write([]byte(`{"id":"created"}`))

					return
				}

				_, _ = w.Write([]byte(`{"keys":[{"id":"created","rsa_public_key":"public-key","description":"Key","is_primary":false}]}`))
			}))
			plan := lifecyclePlan(t, r, brazeSDKAuthenticationKeyModel{ID: types.StringUnknown(), AppID: types.StringValue("app-1"), RSAPublicKey: types.StringValue("public-key"), Description: types.StringValue("Key"), Primary: types.BoolUnknown()})
			response := resource.CreateResponse{State: tfsdk.State{Schema: plan.Schema}, Identity: lifecycleIdentity(t, r)}
			r.Create(t.Context(), resource.CreateRequest{Plan: plan}, &response)

			if status == http.StatusTooManyRequests {
				assert.False(t, response.Diagnostics.HasError(), "%v", response.Diagnostics)
				assert.EqualValues(t, 2, creates.Load())
			} else {
				assert.True(t, response.Diagnostics.HasError())
				assert.EqualValues(t, 1, creates.Load())
			}
		})
	}
}
