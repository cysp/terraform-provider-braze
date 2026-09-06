package testing_test

import (
	"testing"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	brazetesting "github.com/cysp/terraform-provider-braze/internal/braze-client-go/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSDKAuthenticationKeyLifecycle(t *testing.T) {
	t.Parallel()

	handler := brazetesting.NewBrazeHandler()

	firstResponse, err := handler.CreateSDKAuthenticationKey(t.Context(), &brazeclient.CreateSDKAuthenticationKeyRequest{
		AppID:           "app-1",
		RsaPublicKeyStr: "public-key-1",
		Description:     "First key",
		MakePrimary:     brazeclient.NewOptBool(true),
	})
	require.NoError(t, err)

	firstResponseBody := firstResponse.GetResponse()
	firstID := firstResponseBody.GetID()
	require.NotEmpty(t, firstID)

	secondResponse, err := handler.CreateSDKAuthenticationKey(t.Context(), &brazeclient.CreateSDKAuthenticationKeyRequest{
		AppID:           "app-1",
		RsaPublicKeyStr: "public-key-2",
		Description:     "Second key",
	})
	require.NoError(t, err)

	secondResponseBody := secondResponse.GetResponse()
	secondID := secondResponseBody.GetID()
	require.NotEmpty(t, secondID)

	_, err = handler.DeleteSDKAuthenticationKey(t.Context(), &brazeclient.DeleteSDKAuthenticationKeyRequest{
		AppID: "app-1",
		KeyID: firstID,
	})
	require.Error(t, err)

	primaryResponse, err := handler.SetPrimarySDKAuthenticationKey(t.Context(), &brazeclient.SetPrimarySDKAuthenticationKeyRequest{
		AppID: "app-1",
		KeyID: secondID,
	})
	require.NoError(t, err)

	primaryResponseBody := primaryResponse.GetResponse()
	require.Len(t, primaryResponseBody.GetKeys(), 2)

	_, err = handler.DeleteSDKAuthenticationKey(t.Context(), &brazeclient.DeleteSDKAuthenticationKeyRequest{
		AppID: "app-1",
		KeyID: firstID,
	})
	require.NoError(t, err)

	listResponse, err := handler.ListSDKAuthenticationKeys(
		t.Context(),
		brazeclient.ListSDKAuthenticationKeysParams{AppID: "app-1"},
	)
	require.NoError(t, err)

	listResponseBody := listResponse.GetResponse()
	require.Len(t, listResponseBody.GetKeys(), 1)
	assert.Equal(t, secondID, listResponseBody.GetKeys()[0].GetID())
	assert.True(t, listResponseBody.GetKeys()[0].GetIsPrimary())
}

func TestSDKAuthenticationKeyLimit(t *testing.T) {
	t.Parallel()

	handler := brazetesting.NewBrazeHandler()

	for index := range 3 {
		_, err := handler.CreateSDKAuthenticationKey(t.Context(), &brazeclient.CreateSDKAuthenticationKeyRequest{
			AppID:           "app-1",
			RsaPublicKeyStr: "public-key",
			Description:     "Key",
		})
		require.NoError(t, err, "create key %d", index)
	}

	_, err := handler.CreateSDKAuthenticationKey(t.Context(), &brazeclient.CreateSDKAuthenticationKeyRequest{
		AppID:           "app-1",
		RsaPublicKeyStr: "public-key",
		Description:     "One key too many",
	})
	require.Error(t, err)
}
