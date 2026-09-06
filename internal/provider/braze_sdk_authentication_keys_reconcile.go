package provider

import (
	"context"
	"errors"
	"fmt"
	"slices"
)

var (
	errSDKAuthenticationKeysUnexpectedCollection = errors.New("unexpected SDK Authentication key collection after mutation")
	errSDKAuthenticationKeysUnexpectedID         = errors.New("create returned an empty or already known SDK Authentication key ID")
	errSDKAuthenticationKeysUnknownOperation     = errors.New("unknown SDK Authentication key operation")
)

type sdkAuthenticationKeysResult struct {
	Keys            []sdkAuthenticationCollectionKey
	PrimaryVerified bool
	Attempted       bool
}

func reconcileSDKAuthenticationKeys(
	ctx context.Context,
	client sdkAuthenticationKeysClient,
	appID string,
	current, desired []sdkAuthenticationCollectionKey,
) (sdkAuthenticationKeysResult, error) {
	result := sdkAuthenticationKeysResult{Keys: slices.Clone(current), PrimaryVerified: true}

	operations, err := planSDKAuthenticationKeys(current, desired)
	if err != nil {
		return result, err
	}

	for _, operation := range operations {
		result.Attempted = true

		result, err = executeSDKAuthenticationKeysOperation(ctx, client, appID, result, operation)
		if err != nil {
			return result, err
		}
	}

	return result, nil
}

func executeSDKAuthenticationKeysOperation(
	ctx context.Context,
	client sdkAuthenticationKeysClient,
	appID string,
	result sdkAuthenticationKeysResult,
	operation sdkAuthenticationKeysOperation,
) (sdkAuthenticationKeysResult, error) {
	var (
		observed []sdkAuthenticationCollectionKey
		err      error
	)

	switch operation.Kind {
	case sdkAuthenticationKeysCreate:
		return createSDKAuthenticationKeysMember(ctx, client, appID, result, operation.Key)
	case sdkAuthenticationKeysPromote:
		result.PrimaryVerified = false
		observed, err = client.SetPrimary(ctx, appID, operation.Key.ID)
	case sdkAuthenticationKeysDelete:
		observed, err = client.Delete(ctx, appID, operation.Key.ID)
	default:
		return result, errSDKAuthenticationKeysUnknownOperation
	}

	if err != nil {
		return result, fmt.Errorf("%s SDK Authentication key %s: %w", operation.Kind, operation.Key.ID, err)
	}

	expected := expectedSDKAuthenticationKeysCollection(result.Keys, operation)

	return verifySDKAuthenticationKeysResult(result, observed, expected)
}

func createSDKAuthenticationKeysMember(
	ctx context.Context,
	client sdkAuthenticationKeysClient,
	appID string,
	result sdkAuthenticationKeysResult,
	key sdkAuthenticationCollectionKey,
) (sdkAuthenticationKeysResult, error) {
	if key.Primary {
		result.PrimaryVerified = false
	}

	keyID, err := client.Create(ctx, appID, key)
	if err != nil {
		return result, fmt.Errorf("%w; do not replay an ambiguous create request; inspect the remote collection before another apply", err)
	}

	if keyID == "" || slices.ContainsFunc(result.Keys, func(existing sdkAuthenticationCollectionKey) bool { return existing.ID == keyID }) {
		return result, errSDKAuthenticationKeysUnexpectedID
	}

	key.ID = keyID
	expected := expectedSDKAuthenticationKeysCollection(result.Keys, sdkAuthenticationKeysOperation{Kind: sdkAuthenticationKeysCreate, Key: key})
	// Retain the confirmed generated ID and accepted request values even when the
	// verification read fails. Unverified primary-role changes remain null in state.
	result.Keys = expected

	observed, err := client.List(ctx, appID)
	if err != nil {
		return result, fmt.Errorf("created SDK Authentication key %s but verification failed: %w", keyID, err)
	}

	verified, err := verifySDKAuthenticationKeysResult(result, observed, expected)
	if errors.Is(err, errSDKAuthenticationKeysUnexpectedCollection) && !slices.ContainsFunc(observed, func(existing sdkAuthenticationCollectionKey) bool { return existing.ID == keyID }) {
		// A successful list that omits the just-created ID does not erase the
		// confirmed create result. Keep it for explicit recovery and a later read.
		verified.Keys = append(slices.Clone(observed), key)
		verified.PrimaryVerified = false
	}

	return verified, err
}

func expectedSDKAuthenticationKeysCollection(current []sdkAuthenticationCollectionKey, operation sdkAuthenticationKeysOperation) []sdkAuthenticationCollectionKey {
	expected := slices.Clone(current)

	switch operation.Kind {
	case sdkAuthenticationKeysCreate:
		if operation.Key.Primary {
			for i := range expected {
				expected[i].Primary = false
			}
		}

		expected = append(expected, operation.Key)
	case sdkAuthenticationKeysPromote:
		for i := range expected {
			expected[i].Primary = expected[i].ID == operation.Key.ID
		}
	case sdkAuthenticationKeysDelete:
		expected = slices.DeleteFunc(expected, func(key sdkAuthenticationCollectionKey) bool { return key.ID == operation.Key.ID })
	}

	return expected
}

func verifySDKAuthenticationKeysResult(result sdkAuthenticationKeysResult, observed, expected []sdkAuthenticationCollectionKey) (sdkAuthenticationKeysResult, error) {
	if observed == nil {
		return result, errSDKAuthenticationKeysEmptyResponse
	}

	err := validateSDKAuthenticationKeysCurrent(observed)
	if err != nil {
		return result, err
	}

	result.Keys = slices.Clone(observed)

	result.PrimaryVerified = true
	if !sdkAuthenticationKeysCollectionsEqual(observed, expected) {
		return result, fmt.Errorf("%w: the response does not match the expected operation, or another writer changed the collection. No further mutations were attempted; inspect the collection and create a fresh plan", errSDKAuthenticationKeysUnexpectedCollection)
	}

	return result, nil
}
