//nolint:testpackage
package provider

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func configureTestProvider(t *testing.T, configuredProvider *brazeProvider, baseURL, apiKey any, allowDeferral bool) provider.ConfigureResponse {
	t.Helper()

	var schemaResponse provider.SchemaResponse
	configuredProvider.Schema(t.Context(), provider.SchemaRequest{}, &schemaResponse)

	configType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"base_url": tftypes.String, "api_key": tftypes.String}}
	config := tftypes.NewValue(configType, map[string]tftypes.Value{
		"base_url": tftypes.NewValue(tftypes.String, baseURL),
		"api_key":  tftypes.NewValue(tftypes.String, apiKey),
	})

	var response provider.ConfigureResponse
	configuredProvider.Configure(t.Context(), provider.ConfigureRequest{
		Config:             tfsdk.Config{Schema: schemaResponse.Schema, Raw: config},
		ClientCapabilities: provider.ConfigureProviderClientCapabilities{DeferralAllowed: allowDeferral},
	}, &response)

	return response
}

//nolint:paralleltest // Subtests inherit process environment set by the parent.
func TestProviderConfigurationRejectsUnusableValues(t *testing.T) {
	t.Setenv("BRAZE_API_KEY", "")

	for name, values := range map[string][2]any{
		"missing endpoint":   {nil, "secret"},
		"missing key":        {"https://rest.test.braze.com", nil},
		"empty endpoint":     {"", "secret"},
		"empty key":          {"https://rest.test.braze.com", ""},
		"relative endpoint":  {"rest.test.braze.com", "secret"},
		"unsupported scheme": {"ftp://rest.test.braze.com", "secret"},
		"userinfo":           {"https://username:secret@rest.test.braze.com", "secret"},
		"query":              {"https://rest.test.braze.com?key=secret", "secret"},
		"fragment":           {"https://rest.test.braze.com/#secret", "secret"},
		"unknown endpoint":   {tftypes.UnknownValue, "secret"},
		"unknown key":        {"https://rest.test.braze.com", tftypes.UnknownValue},
	} {
		t.Run(name, func(t *testing.T) {
			response := configureTestProvider(t, NewBrazeProvider("test"), values[0], values[1], false)
			assert.True(t, response.Diagnostics.HasError())
			assert.Nil(t, response.ResourceData)
			assert.Nil(t, response.ListResourceData)

			for _, diagnostic := range response.Diagnostics {
				assert.NotContains(t, diagnostic.Detail(), "secret")
			}
		})
	}
}

func TestProviderConfigurationPrecedence(t *testing.T) {
	for name, test := range map[string]struct {
		configured            any
		environment, expected string
	}{
		"explicit beats environment": {"explicit", "environment", "explicit"},
		"environment":                {nil, "environment", "environment"},
		"injected fallback":          {nil, "", "injected"},
		"empty is explicit":          {"", "environment", ""},
		"unknown stays unknown":      {tftypes.UnknownValue, "environment", ""},
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("BRAZE_API_KEY", test.environment)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "Bearer "+test.expected, r.Header.Get("Authorization"))
				assert.Equal(t, "/catalogs", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"catalogs":[],"message":"success"}`))
			}))
			t.Cleanup(server.Close)
			configuredProvider := NewBrazeProvider("test", WithBaseURL(server.URL), WithAPIKey("injected"), WithHTTPClient(server.Client()))

			response := configureTestProvider(t, configuredProvider, nil, test.configured, false)
			if test.expected == "" {
				assert.True(t, response.Diagnostics.HasError())
				assert.Nil(t, response.ResourceData)

				return
			}

			require.False(t, response.Diagnostics.HasError(), "%v", response.Diagnostics)
			data, ok := response.ResourceData.(brazeProviderData)
			require.True(t, ok)
			assert.Empty(t, collectListForTest(t, data.catalogs.List(t.Context())))
		})
	}
}

func TestProviderConfigurationDefersUnknownValues(t *testing.T) {
	t.Setenv("BRAZE_API_KEY", "environment")

	for _, values := range [][2]any{{tftypes.UnknownValue, "configured"}, {"https://rest.test.braze.com", tftypes.UnknownValue}} {
		response := configureTestProvider(t, NewBrazeProvider("test"), values[0], values[1], true)
		require.NotNil(t, response.Deferred)
		assert.Equal(t, provider.DeferredReasonProviderConfigUnknown, response.Deferred.Reason)
		assert.False(t, response.Diagnostics.HasError())
		assert.Nil(t, response.ResourceData)
	}
}
