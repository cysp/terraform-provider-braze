package provider

import (
	"context"
	"fmt"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type sdkAuthenticationKeysClient interface {
	List(ctx context.Context, appID string) ([]sdkAuthenticationCollectionKey, error)
	Create(ctx context.Context, appID string, key sdkAuthenticationCollectionKey) (string, error)
	SetPrimary(ctx context.Context, appID, keyID string) ([]sdkAuthenticationCollectionKey, error)
	Delete(ctx context.Context, appID, keyID string) ([]sdkAuthenticationCollectionKey, error)
}

type generatedSDKAuthenticationKeysClient struct {
	client *brazeclient.Client
}

var _ sdkAuthenticationKeysClient = generatedSDKAuthenticationKeysClient{}

func newGeneratedSDKAuthenticationKeysClient(client *brazeclient.Client) generatedSDKAuthenticationKeysClient {
	return generatedSDKAuthenticationKeysClient{client: client}
}

func (c generatedSDKAuthenticationKeysClient) List(
	ctx context.Context,
	appID string,
) ([]sdkAuthenticationCollectionKey, error) {
	response, err := c.client.ListSDKAuthenticationKeys(
		ctx,
		brazeclient.ListSDKAuthenticationKeysParams{AppID: appID},
	)

	tflog.Debug(ctx, "braze_sdk_authentication_keys.list", map[string]any{
		"app_id": appID,
	})

	if err != nil {
		return nil, fmt.Errorf("list SDK Authentication keys: %w", err)
	}

	return sdkAuthenticationCollectionFromResponse(response)
}

func (c generatedSDKAuthenticationKeysClient) Create(
	ctx context.Context,
	appID string,
	key sdkAuthenticationCollectionKey,
) (string, error) {
	request := brazeclient.CreateSDKAuthenticationKeyRequest{
		AppID:           appID,
		RsaPublicKeyStr: key.RSAPublicKey,
		Description:     key.Description,
		MakePrimary:     brazeclient.NewOptBool(key.Primary),
	}
	response, err := c.client.CreateSDKAuthenticationKey(ctx, &request)

	tflog.Debug(ctx, "braze_sdk_authentication_keys.create", map[string]any{
		"app_id": appID,
	})

	if err != nil {
		return "", fmt.Errorf("create SDK Authentication key: %w", err)
	}

	if response == nil {
		return "", errSDKAuthenticationKeysEmptyResponse
	}

	responseBody := response.GetResponse()

	keyID := responseBody.GetID()
	if keyID == "" {
		return "", errSDKAuthenticationKeysEmptyID
	}

	return keyID, nil
}

func (c generatedSDKAuthenticationKeysClient) SetPrimary(
	ctx context.Context,
	appID string,
	keyID string,
) ([]sdkAuthenticationCollectionKey, error) {
	request := brazeclient.SetPrimarySDKAuthenticationKeyRequest{AppID: appID, KeyID: keyID}
	response, err := c.client.SetPrimarySDKAuthenticationKey(ctx, &request)

	tflog.Debug(ctx, "braze_sdk_authentication_keys.set_primary", map[string]any{
		"app_id": appID,
		"key_id": keyID,
	})

	if err != nil {
		return nil, fmt.Errorf("set primary SDK Authentication key: %w", err)
	}

	return sdkAuthenticationCollectionFromResponse(response)
}

func (c generatedSDKAuthenticationKeysClient) Delete(
	ctx context.Context,
	appID string,
	keyID string,
) ([]sdkAuthenticationCollectionKey, error) {
	request := brazeclient.DeleteSDKAuthenticationKeyRequest{AppID: appID, KeyID: keyID}
	response, err := c.client.DeleteSDKAuthenticationKey(ctx, &request)

	tflog.Debug(ctx, "braze_sdk_authentication_keys.delete", map[string]any{
		"app_id": appID,
		"key_id": keyID,
	})

	if err != nil {
		return nil, fmt.Errorf("delete SDK Authentication key: %w", err)
	}

	return sdkAuthenticationCollectionFromResponse(response)
}

func sdkAuthenticationCollectionFromResponse(
	response *brazeclient.SDKAuthenticationKeysResponseStatusCode,
) ([]sdkAuthenticationCollectionKey, error) {
	keys, err := sdkAuthenticationKeysFromResponse(response)
	if err != nil {
		return nil, err
	}

	result := make([]sdkAuthenticationCollectionKey, 0, len(keys))

	for _, key := range keys {
		result = append(result, sdkAuthenticationCollectionKey{
			ID:           key.GetID(),
			RSAPublicKey: key.GetRsaPublicKey(),
			Description:  key.GetDescription(),
			Primary:      key.GetIsPrimary(),
		})
	}

	return result, nil
}
