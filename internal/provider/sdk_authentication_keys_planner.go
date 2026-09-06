package provider

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
)

var (
	errSDKAuthenticationKeysInvalidCollection = errors.New("invalid SDK Authentication key collection")
	errSDKAuthenticationKeysTemporaryPrimary  = errors.New("SDK Authentication key transition requires a temporary primary")
)

type sdkAuthenticationCollectionKey struct {
	ID           string
	RSAPublicKey string
	Description  string
	Primary      bool
}

type sdkAuthenticationKeysOperationKind string

const (
	sdkAuthenticationKeysCreate  sdkAuthenticationKeysOperationKind = "create"
	sdkAuthenticationKeysPromote sdkAuthenticationKeysOperationKind = "promote"
	sdkAuthenticationKeysDelete  sdkAuthenticationKeysOperationKind = "delete"
)

type sdkAuthenticationKeysOperation struct {
	Kind sdkAuthenticationKeysOperationKind
	Key  sdkAuthenticationCollectionKey
}

// planSDKAuthenticationKeys simulates the entire transition before returning any
// operations. Configured IDs have no bearing on matching or operation targets.
func planSDKAuthenticationKeys(current, desired []sdkAuthenticationCollectionKey) ([]sdkAuthenticationKeysOperation, error) {
	err := validateSDKAuthenticationKeysCurrent(current)
	if err != nil {
		return nil, err
	}

	err = validateSDKAuthenticationKeysUniqueMaterial(current)
	if err != nil {
		return nil, fmt.Errorf("observed collection: %w", err)
	}

	err = validateSDKAuthenticationKeysDesired(desired)
	if err != nil {
		return nil, err
	}

	observed := slices.Clone(current)
	slices.SortFunc(observed, func(a, b sdkAuthenticationCollectionKey) int { return cmp.Compare(a.ID, b.ID) })

	configured := slices.Clone(desired)
	slices.SortFunc(configured, compareSDKAuthenticationKeysDesired)

	var operations []sdkAuthenticationKeysOperation

	primary := configured[0]
	if index := sdkAuthenticationKeysMaterialIndex(observed, primary); index >= 0 && !observed[index].Primary {
		for i := range observed {
			observed[i].Primary = i == index
		}

		operations = append(operations, sdkAuthenticationKeysOperation{Kind: sdkAuthenticationKeysPromote, Key: observed[index]})
	}

	for _, key := range configured {
		if sdkAuthenticationKeysMaterialIndex(observed, key) >= 0 {
			continue
		}

		if len(observed) == maxSDKAuthenticationKeys {
			index := sdkAuthenticationKeysRemovableIndex(observed, configured)
			if index < 0 {
				return nil, sdkAuthenticationKeysCapacityError(observed)
			}

			operations = append(operations, sdkAuthenticationKeysOperation{Kind: sdkAuthenticationKeysDelete, Key: observed[index]})
			observed = slices.Delete(observed, index, index+1)
		}

		key.ID = ""
		if key.Primary {
			for i := range observed {
				observed[i].Primary = false
			}
		}

		operations = append(operations, sdkAuthenticationKeysOperation{Kind: sdkAuthenticationKeysCreate, Key: key})
		observed = append(observed, key)
	}

	for _, key := range observed {
		if sdkAuthenticationKeysMaterialIndex(configured, key) < 0 {
			operations = append(operations, sdkAuthenticationKeysOperation{Kind: sdkAuthenticationKeysDelete, Key: key})
		}
	}

	return operations, nil
}

// validateSDKAuthenticationKeysCurrent permits ambiguous immutable pairs so Read
// can preserve every returned ID. Reconciliation checks that ambiguity separately.
func validateSDKAuthenticationKeysCurrent(keys []sdkAuthenticationCollectionKey) error {
	if len(keys) > maxSDKAuthenticationKeys {
		return fmt.Errorf("%w: observed collection has %d keys; at most three are supported", errSDKAuthenticationKeysInvalidCollection, len(keys))
	}

	ids := make(map[string]struct{}, len(keys))
	primaryCount := 0

	for _, key := range keys {
		if key.ID == "" {
			return fmt.Errorf("%w: observed key has an empty ID", errSDKAuthenticationKeysInvalidCollection)
		}

		if _, exists := ids[key.ID]; exists {
			return fmt.Errorf("%w: observed key ID %q occurs more than once", errSDKAuthenticationKeysInvalidCollection, key.ID)
		}

		ids[key.ID] = struct{}{}
		if key.Primary {
			primaryCount++
		}
	}

	if primaryCount > 1 {
		return fmt.Errorf("%w: observed collection has %d primary keys", errSDKAuthenticationKeysInvalidCollection, primaryCount)
	}

	return nil
}

func validateSDKAuthenticationKeysDesired(keys []sdkAuthenticationCollectionKey) error {
	if len(keys) < 1 || len(keys) > maxSDKAuthenticationKeys {
		return fmt.Errorf("%w: configure one to three keys", errSDKAuthenticationKeysInvalidCollection)
	}

	primaryCount := 0

	for _, key := range keys {
		if key.RSAPublicKey == "" || key.Description == "" {
			return fmt.Errorf("%w: configured rsa_public_key and description must both be nonempty", errSDKAuthenticationKeysInvalidCollection)
		}

		if key.Primary {
			primaryCount++
		}
	}

	if primaryCount != 1 {
		return fmt.Errorf("%w: configure exactly one primary key; got %d", errSDKAuthenticationKeysInvalidCollection, primaryCount)
	}

	err := validateSDKAuthenticationKeysUniqueMaterial(keys)
	if err != nil {
		return fmt.Errorf("configured collection: %w", err)
	}

	return nil
}

func validateSDKAuthenticationKeysUniqueMaterial(keys []sdkAuthenticationCollectionKey) error {
	for i, key := range keys {
		if sdkAuthenticationKeysMaterialIndex(keys[:i], key) >= 0 {
			return fmt.Errorf("%w: duplicate (rsa_public_key, description) pairs make reconciliation ambiguous; resolve the duplicate members before applying", errSDKAuthenticationKeysInvalidCollection)
		}
	}

	return nil
}

func sameSDKAuthenticationKeyMaterial(a, b sdkAuthenticationCollectionKey) bool {
	return a.RSAPublicKey == b.RSAPublicKey && a.Description == b.Description
}

func sdkAuthenticationKeysMaterialIndex(keys []sdkAuthenticationCollectionKey, key sdkAuthenticationCollectionKey) int {
	return slices.IndexFunc(keys, func(candidate sdkAuthenticationCollectionKey) bool {
		return sameSDKAuthenticationKeyMaterial(candidate, key)
	})
}

func compareSDKAuthenticationKeysDesired(left, right sdkAuthenticationCollectionKey) int {
	if left.Primary != right.Primary {
		if left.Primary {
			return -1
		}

		return 1
	}

	return cmp.Or(cmp.Compare(left.RSAPublicKey, right.RSAPublicKey), cmp.Compare(left.Description, right.Description))
}

func sdkAuthenticationKeysRemovableIndex(current, desired []sdkAuthenticationCollectionKey) int {
	return slices.IndexFunc(current, func(key sdkAuthenticationCollectionKey) bool {
		return !key.Primary && sdkAuthenticationKeysMaterialIndex(desired, key) < 0
	})
}

func sdkAuthenticationKeysCapacityError(current []sdkAuthenticationCollectionKey) error {
	for _, key := range current {
		if key.Primary {
			return fmt.Errorf("%w: all three slots are occupied and current primary %q is the only unwanted key; it cannot be deleted to create the configured primary. Explicitly promote a surviving key in a preceding apply, or retire another key to free a slot", errSDKAuthenticationKeysTemporaryPrimary, key.ID)
		}
	}

	return fmt.Errorf("%w: no unwanted non-primary key is available to free a slot", errSDKAuthenticationKeysTemporaryPrimary)
}

func sdkAuthenticationKeysCollectionsEqual(a, b []sdkAuthenticationCollectionKey) bool {
	compare := func(a, b sdkAuthenticationCollectionKey) int {
		return cmp.Or(cmp.Compare(a.ID, b.ID), compareSDKAuthenticationKeysDesired(a, b))
	}

	left, right := slices.Clone(a), slices.Clone(b)
	slices.SortFunc(left, compare)
	slices.SortFunc(right, compare)

	return slices.Equal(left, right)
}
