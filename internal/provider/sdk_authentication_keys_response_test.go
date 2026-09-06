//nolint:testpackage
package provider

import (
	"testing"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSDKAuthenticationKeysFromResponseRejectsAbsentResponses(t *testing.T) {
	t.Parallel()

	for name, test := range map[string]struct {
		response *brazeclient.SDKAuthenticationKeysResponseStatusCode
		wantErr  error
	}{
		"nil response": {nil, errSDKAuthenticationKeysEmptyResponse},
		"missing keys": {&brazeclient.SDKAuthenticationKeysResponseStatusCode{}, errSDKAuthenticationKeysMissingKeys},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			keys, err := sdkAuthenticationKeysFromResponse(test.response)

			require.ErrorIs(t, err, test.wantErr)
			assert.Nil(t, keys)
			assert.False(t, isBrazeObjectNotFound(err))
		})
	}
}

func TestSDKAuthenticationKeysFromResponsePreservesObservedCollections(t *testing.T) {
	t.Parallel()

	for name, observed := range map[string][]brazeclient.SDKAuthenticationKey{
		"empty": {},
		"zero primary and duplicate material": {
			{ID: "key-2", RsaPublicKey: "shared-key", Description: "", IsPrimary: false},
			{ID: "key-1", RsaPublicKey: "shared-key", Description: "", IsPrimary: false},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			keys, err := sdkAuthenticationKeysFromResponse(&brazeclient.SDKAuthenticationKeysResponseStatusCode{
				Response: brazeclient.SDKAuthenticationKeysResponse{Keys: observed},
			})

			require.NoError(t, err)
			assert.NotNil(t, keys)
			assert.Equal(t, observed, keys)
		})
	}
}
