package provider

import (
	"context"
	"errors"
	"fmt"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	errSDKAuthenticationKeyEmptyResponse = errors.New("empty SDK Authentication key response")
	errSDKAuthenticationKeyNotFound      = errors.New("SDK Authentication key not found")
	errSDKAuthenticationKeyNotPrimary    = errors.New("SDK Authentication key was not made primary")
	errSDKAuthenticationMultiplePrimary  = errors.New("multiple primary SDK Authentication keys returned")
	errSDKAuthenticationPrimaryDelete    = errors.New("the primary SDK Authentication key cannot be deleted")
	errSDKAuthenticationKeyStillExists   = errors.New("deleted SDK Authentication key remains in response")
)

type sdkAuthenticationKeyCreateResult struct {
	Key               brazeSDKAuthenticationKeyModel
	VerificationError error
}

type sdkAuthenticationKeyClient interface {
	Create(ctx context.Context, plan brazeSDKAuthenticationKeyModel) (sdkAuthenticationKeyCreateResult, error)
	Read(ctx context.Context, appID, keyID string) (brazeSDKAuthenticationKeyModel, error)
	SetPrimary(ctx context.Context, appID, keyID string) (brazeSDKAuthenticationKeyModel, error)
	Delete(ctx context.Context, appID, keyID string) error
}

type generatedSDKAuthenticationKeyClient struct {
	client *brazeclient.Client
}

func newGeneratedSDKAuthenticationKeyClient(client *brazeclient.Client) generatedSDKAuthenticationKeyClient {
	return generatedSDKAuthenticationKeyClient{client: client}
}

func (c generatedSDKAuthenticationKeyClient) Create(
	ctx context.Context,
	plan brazeSDKAuthenticationKeyModel,
) (sdkAuthenticationKeyCreateResult, error) {
	request := brazeclient.CreateSDKAuthenticationKeyRequest{
		AppID:           plan.AppID.ValueString(),
		RsaPublicKeyStr: plan.RSAPublicKey.ValueString(),
		Description:     plan.Description.ValueString(),
	}
	if !plan.Primary.IsNull() && !plan.Primary.IsUnknown() {
		request.MakePrimary = brazeclient.NewOptBool(plan.Primary.ValueBool())
	}

	response, err := c.client.CreateSDKAuthenticationKey(ctx, &request)

	tflog.Info(ctx, "braze_sdk_authentication_key.create", map[string]any{
		"app_id":   request.AppID,
		"response": response,
		"err":      err,
	})

	if err != nil {
		return sdkAuthenticationKeyCreateResult{}, fmt.Errorf("create SDK Authentication key: %w", err)
	}

	if response == nil {
		return sdkAuthenticationKeyCreateResult{}, errSDKAuthenticationKeyEmptyResponse
	}

	responseBody := response.GetResponse()
	if responseBody.GetID() == "" {
		return sdkAuthenticationKeyCreateResult{}, errSDKAuthenticationKeyEmptyResponse
	}

	provisional := plan

	provisional.ID = types.StringValue(responseBody.GetID())
	if plan.Primary.IsNull() || plan.Primary.IsUnknown() {
		provisional.Primary = types.BoolNull()
	}

	key, verificationErr := c.Read(ctx, request.AppID, responseBody.GetID())
	if verificationErr != nil {
		return sdkAuthenticationKeyCreateResult{
			Key:               provisional,
			VerificationError: fmt.Errorf("verify created SDK Authentication key: %w", verificationErr),
		}, nil
	}

	if !plan.Primary.IsNull() && !plan.Primary.IsUnknown() && plan.Primary.ValueBool() && !key.Primary.ValueBool() {
		return sdkAuthenticationKeyCreateResult{
			Key:               provisional,
			VerificationError: errSDKAuthenticationKeyNotPrimary,
		}, nil
	}

	return sdkAuthenticationKeyCreateResult{Key: key}, nil
}

func (c generatedSDKAuthenticationKeyClient) Read(
	ctx context.Context,
	appID string,
	keyID string,
) (brazeSDKAuthenticationKeyModel, error) {
	response, err := c.client.ListSDKAuthenticationKeys(ctx, brazeclient.ListSDKAuthenticationKeysParams{AppID: appID})

	tflog.Info(ctx, "braze_sdk_authentication_key.read", map[string]any{
		"app_id":   appID,
		"key_id":   keyID,
		"response": response,
		"err":      err,
	})

	if err != nil {
		return brazeSDKAuthenticationKeyModel{}, fmt.Errorf("list SDK Authentication keys: %w", err)
	}

	if response == nil {
		return brazeSDKAuthenticationKeyModel{}, errSDKAuthenticationKeyEmptyResponse
	}

	responseBody := response.GetResponse()

	key, err := findSDKAuthenticationKey(responseBody.GetKeys(), keyID)
	if err != nil {
		return brazeSDKAuthenticationKeyModel{}, err
	}

	return newBrazeSDKAuthenticationKeyModel(appID, key), nil
}

func (c generatedSDKAuthenticationKeyClient) SetPrimary(
	ctx context.Context,
	appID string,
	keyID string,
) (brazeSDKAuthenticationKeyModel, error) {
	request := brazeclient.SetPrimarySDKAuthenticationKeyRequest{AppID: appID, KeyID: keyID}
	response, err := c.client.SetPrimarySDKAuthenticationKey(ctx, &request)

	tflog.Info(ctx, "braze_sdk_authentication_key.set_primary", map[string]any{
		"app_id":   appID,
		"key_id":   keyID,
		"response": response,
		"err":      err,
	})

	if err != nil {
		return brazeSDKAuthenticationKeyModel{}, fmt.Errorf("set primary SDK Authentication key: %w", err)
	}

	if response == nil {
		return brazeSDKAuthenticationKeyModel{}, errSDKAuthenticationKeyEmptyResponse
	}

	responseBody := response.GetResponse()
	keys := responseBody.GetKeys()

	key, err := findSDKAuthenticationKey(keys, keyID)
	if err != nil {
		return brazeSDKAuthenticationKeyModel{}, fmt.Errorf("validate primary SDK Authentication key response: %w", err)
	}

	if !key.GetIsPrimary() {
		return brazeSDKAuthenticationKeyModel{}, fmt.Errorf("%w: %s", errSDKAuthenticationKeyNotPrimary, keyID)
	}

	primaryCount := 0

	for _, candidate := range keys {
		if candidate.GetIsPrimary() {
			primaryCount++
		}
	}

	if primaryCount != 1 {
		return brazeSDKAuthenticationKeyModel{}, fmt.Errorf("%w: got %d", errSDKAuthenticationMultiplePrimary, primaryCount)
	}

	return newBrazeSDKAuthenticationKeyModel(appID, key), nil
}

func (c generatedSDKAuthenticationKeyClient) Delete(ctx context.Context, appID, keyID string) error {
	request := brazeclient.DeleteSDKAuthenticationKeyRequest{AppID: appID, KeyID: keyID}
	response, err := c.client.DeleteSDKAuthenticationKey(ctx, &request)

	tflog.Info(ctx, "braze_sdk_authentication_key.delete", map[string]any{
		"app_id":   appID,
		"key_id":   keyID,
		"response": response,
		"err":      err,
	})

	if err != nil {
		key, readErr := c.Read(ctx, appID, keyID)
		if isBrazeObjectNotFound(readErr) {
			return nil
		}

		if readErr == nil && key.Primary.ValueBool() {
			return fmt.Errorf("%w: %w", errSDKAuthenticationPrimaryDelete, err)
		}

		if readErr != nil {
			return fmt.Errorf("delete SDK Authentication key: %w; verify deletion: %w", err, readErr)
		}

		return fmt.Errorf("delete SDK Authentication key: %w", err)
	}

	if response == nil {
		return errSDKAuthenticationKeyEmptyResponse
	}

	responseBody := response.GetResponse()

	_, findErr := findSDKAuthenticationKey(responseBody.GetKeys(), keyID)
	if findErr == nil {
		return fmt.Errorf("%w: %s", errSDKAuthenticationKeyStillExists, keyID)
	} else if !isBrazeObjectNotFound(findErr) {
		return fmt.Errorf("validate deleted SDK Authentication key response: %w", findErr)
	}

	return nil
}

func findSDKAuthenticationKey(keys []brazeclient.SDKAuthenticationKey, keyID string) (brazeclient.SDKAuthenticationKey, error) {
	for _, key := range keys {
		if key.GetID() == keyID {
			return key, nil
		}
	}

	return brazeclient.SDKAuthenticationKey{}, brazeObjectNotFoundError{
		err: fmt.Errorf("%w: %s", errSDKAuthenticationKeyNotFound, keyID),
	}
}
