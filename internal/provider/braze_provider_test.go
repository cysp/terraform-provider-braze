package provider_test

import (
	"fmt"
	"os"
	"testing"

	. "github.com/cysp/terraform-provider-braze/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeTestAccProtoV6ProviderFactories(options ...BrazeProviderOption) map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"braze": providerserver.NewProtocol6WithError(NewBrazeProvider("test", options...)),
	}
}

//nolint:gochecknoglobals
var testAccProtoV6ProviderFactories = makeTestAccProtoV6ProviderFactories()

func providerConfigDynamicValue(config map[string]any) (tfprotov6.DynamicValue, error) {
	providerConfigTypes := map[string]tftypes.Type{
		"base_url": tftypes.String,
		"api_key":  tftypes.String,
	}
	providerConfigObjectType := tftypes.Object{AttributeTypes: providerConfigTypes}

	providerConfigObjectValue := tftypes.NewValue(providerConfigObjectType, map[string]tftypes.Value{
		"base_url": tftypes.NewValue(tftypes.String, config["base_url"]),
		"api_key":  tftypes.NewValue(tftypes.String, config["api_key"]),
	})

	value, err := tfprotov6.NewDynamicValue(providerConfigObjectType, providerConfigObjectValue)
	if err != nil {
		err = fmt.Errorf("failed to create dynamic value: %w", err)
	}

	return value, err
}

func TestProtocol6ProviderServerSchemaVersion(t *testing.T) {
	t.Parallel()

	providerServer, err := testAccProtoV6ProviderFactories["braze"]()
	require.NotNil(t, providerServer)
	require.NoError(t, err)

	resp, err := providerServer.GetProviderSchema(t.Context(), &tfprotov6.GetProviderSchemaRequest{})
	require.NotNil(t, resp.Provider)
	require.NoError(t, err)
	assert.Empty(t, resp.Diagnostics)

	assert.EqualValues(t, 0, resp.Provider.Version)
}

func TestProtocol6ProviderServerConfigure(t *testing.T) {
	if os.Getenv("TF_ACC") != "" {
		return
	}

	tests := map[string]struct {
		config          map[string]any
		env             map[string]string
		expectedSuccess bool
	}{
		"config: base_url": {
			config: map[string]any{
				"base_url": "https://rest.test.braze.com",
			},
			expectedSuccess: true,
		},
		"config: api_key": {
			config: map[string]any{
				"api_key": "CFPAT-12345",
			},
			expectedSuccess: true,
		},
		"config: base_url,api_key": {
			config: map[string]any{
				"base_url": "https://rest.test.braze.com",
				"api_key":  "CFPAT-12345",
			},
			expectedSuccess: true,
		},
		"config: base_url(invalid),api_key": {
			config: map[string]any{
				"base_url": "url://an invalid url %/",
				"api_key":  "CFPAT-12345",
			},
			expectedSuccess: false,
		},
		"config: base_url env: api_key": {
			config: map[string]any{
				"base_url": "https://rest.test.braze.com",
			},
			env: map[string]string{
				"BRAZE_API_KEY": "CFPAT-12345",
			},
			expectedSuccess: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for key, value := range test.env {
				t.Setenv(key, value)
			}

			providerServer, err := testAccProtoV6ProviderFactories["braze"]()
			require.NotNil(t, providerServer)
			require.NoError(t, err)

			providerConfigValue, err := providerConfigDynamicValue(test.config)
			require.NotNil(t, providerConfigValue)
			require.NoError(t, err)

			resp, err := providerServer.ConfigureProvider(t.Context(), &tfprotov6.ConfigureProviderRequest{
				Config: &providerConfigValue,
			})
			require.NotNil(t, resp)
			require.NoError(t, err)

			if test.expectedSuccess {
				assert.Empty(t, resp.Diagnostics)
			} else {
				assert.NotEmpty(t, resp.Diagnostics)
			}
		})
	}
}
