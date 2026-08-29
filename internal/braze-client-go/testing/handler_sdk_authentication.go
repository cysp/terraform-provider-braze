package testing

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	"github.com/google/uuid"
)

const maxSDKAuthenticationKeysPerApp = 3

var (
	errSDKAuthenticationKeyLimitReached = errors.New("an app can have up to three SDK Authentication keys")
	errSDKAuthenticationKeyNotFound     = errors.New("SDK Authentication key not found")
	errSDKAuthenticationPrimaryDelete   = errors.New("the primary SDK Authentication key cannot be deleted")
	errSDKAuthenticationDescription     = errors.New("SDK Authentication key description cannot be empty")
)

func (h *Handler) CreateSDKAuthenticationKey(
	_ context.Context,
	req *brazeclient.CreateSDKAuthenticationKeyRequest,
) (*brazeclient.CreateSDKAuthenticationKeyResponseStatusCode, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Braze requires a valid RSA public key but does not document its exact
	// accepted PEM encodings. The test server stores the supplied public value
	// without imposing a narrower parser than the service contract.
	if strings.TrimSpace(req.Description) == "" {
		return nil, fmt.Errorf("%w: %w", newStatusCodeError(http.StatusBadRequest), errSDKAuthenticationDescription)
	}

	keys := h.sdkAuthenticationKeys[req.AppID]
	if len(keys) >= maxSDKAuthenticationKeysPerApp {
		return nil, fmt.Errorf("%w: %w", newStatusCodeError(http.StatusBadRequest), errSDKAuthenticationKeyLimitReached)
	}

	if keys == nil {
		keys = make(map[string]brazeclient.SDKAuthenticationKey)
		h.sdkAuthenticationKeys[req.AppID] = keys
	}

	keyID := uuid.NewString()

	// Braze does not document whether the first key becomes primary when
	// make_primary is omitted. The test server deterministically honors only an
	// explicit true value; provider tests must not treat that as service fact.
	makePrimary := req.MakePrimary.Or(false)
	if makePrimary {
		clearPrimarySDKAuthenticationKey(keys)
	}

	keys[keyID] = brazeclient.SDKAuthenticationKey{
		ID:           keyID,
		RsaPublicKey: req.RsaPublicKeyStr,
		Description:  req.Description,
		IsPrimary:    makePrimary,
	}

	return &brazeclient.CreateSDKAuthenticationKeyResponseStatusCode{
		StatusCode: http.StatusOK,
		Response:   brazeclient.CreateSDKAuthenticationKeyResponse{ID: keyID},
	}, nil
}

func (h *Handler) ListSDKAuthenticationKeys(
	_ context.Context,
	params brazeclient.ListSDKAuthenticationKeysParams,
) (*brazeclient.SDKAuthenticationKeysResponseStatusCode, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	return sdkAuthenticationKeysResponse(h.sdkAuthenticationKeys[params.AppID]), nil
}

func (h *Handler) SetPrimarySDKAuthenticationKey(
	_ context.Context,
	req *brazeclient.SetPrimarySDKAuthenticationKeyRequest,
) (*brazeclient.SDKAuthenticationKeysResponseStatusCode, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	keys := h.sdkAuthenticationKeys[req.AppID]

	key, ok := keys[req.KeyID]
	if !ok {
		// Braze does not document the endpoint-specific status for a missing key.
		// The test server uses 404 as a deterministic local convention.
		return nil, fmt.Errorf("%w: %w", newStatusCodeError(http.StatusNotFound), errSDKAuthenticationKeyNotFound)
	}

	clearPrimarySDKAuthenticationKey(keys)

	key.IsPrimary = true
	keys[req.KeyID] = key

	return sdkAuthenticationKeysResponse(keys), nil
}

func (h *Handler) DeleteSDKAuthenticationKey(
	_ context.Context,
	req *brazeclient.DeleteSDKAuthenticationKeyRequest,
) (*brazeclient.SDKAuthenticationKeysResponseStatusCode, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	keys := h.sdkAuthenticationKeys[req.AppID]

	key, ok := keys[req.KeyID]
	if !ok {
		// Braze does not document the endpoint-specific status for a missing key.
		// The test server uses 404 as a deterministic local convention.
		return nil, fmt.Errorf("%w: %w", newStatusCodeError(http.StatusNotFound), errSDKAuthenticationKeyNotFound)
	}

	if key.IsPrimary {
		return nil, fmt.Errorf("%w: %w", newStatusCodeError(http.StatusBadRequest), errSDKAuthenticationPrimaryDelete)
	}

	delete(keys, req.KeyID)

	return sdkAuthenticationKeysResponse(keys), nil
}

func clearPrimarySDKAuthenticationKey(keys map[string]brazeclient.SDKAuthenticationKey) {
	for id, key := range keys {
		key.IsPrimary = false
		keys[id] = key
	}
}

func sdkAuthenticationKeysResponse(keys map[string]brazeclient.SDKAuthenticationKey) *brazeclient.SDKAuthenticationKeysResponseStatusCode {
	ids := make([]string, 0, len(keys))
	for id := range keys {
		ids = append(ids, id)
	}

	sort.Strings(ids)

	result := make([]brazeclient.SDKAuthenticationKey, 0, len(ids))
	for _, id := range ids {
		result = append(result, keys[id])
	}

	return &brazeclient.SDKAuthenticationKeysResponseStatusCode{
		StatusCode: http.StatusOK,
		Response:   brazeclient.SDKAuthenticationKeysResponse{Keys: result},
	}
}
