package provider

import (
	"errors"
	"fmt"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
)

const maxSDKAuthenticationKeys = 3

var (
	errSDKAuthenticationKeysEmptyResponse   = errors.New("empty SDK Authentication keys response")
	errSDKAuthenticationKeysMissingKeys     = errors.New("SDK Authentication keys response is missing keys")
	errSDKAuthenticationKeysEmptyID         = errors.New("SDK Authentication key response contains an empty ID")
	errSDKAuthenticationKeysDuplicateID     = errors.New("SDK Authentication key response contains a duplicate ID")
	errSDKAuthenticationKeysTooMany         = errors.New("SDK Authentication keys response contains more than three keys")
	errSDKAuthenticationKeysMultiplePrimary = errors.New("SDK Authentication keys response contains multiple primary keys")
)

// sdkAuthenticationKeysFromResponse validates collection structure before a
// resource interprets identity, primary status, or absence. Empty collections,
// zero primary keys, and duplicate immutable material remain observable.
func sdkAuthenticationKeysFromResponse(
	response *brazeclient.SDKAuthenticationKeysResponseStatusCode,
) ([]brazeclient.SDKAuthenticationKey, error) {
	if response == nil {
		return nil, errSDKAuthenticationKeysEmptyResponse
	}

	responseBody := response.GetResponse()

	keys := responseBody.GetKeys()
	if keys == nil {
		return nil, errSDKAuthenticationKeysMissingKeys
	}

	if len(keys) > maxSDKAuthenticationKeys {
		return nil, fmt.Errorf("%w: got %d", errSDKAuthenticationKeysTooMany, len(keys))
	}

	seenIDs := make(map[string]struct{}, len(keys))
	primaryCount := 0

	for _, key := range keys {
		keyID := key.GetID()
		if keyID == "" {
			return nil, errSDKAuthenticationKeysEmptyID
		}

		if _, exists := seenIDs[keyID]; exists {
			return nil, fmt.Errorf("%w: %s", errSDKAuthenticationKeysDuplicateID, keyID)
		}

		seenIDs[keyID] = struct{}{}

		if key.GetIsPrimary() {
			primaryCount++
		}
	}

	if primaryCount > 1 {
		return nil, fmt.Errorf("%w: got %d", errSDKAuthenticationKeysMultiplePrimary, primaryCount)
	}

	return keys, nil
}
