package provider //nolint:testpackage // The pure planner is deliberately package-private.

import (
	"fmt"
	"math/bits"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The oracle represents keys as six independent symbols and a primary index.
// It searches API-legal transitions without using planner matching, ordering,
// validation, or simulation helpers. Six symbols cover disjoint full collections.
type sdkKeysOracleState struct {
	members uint
	primary int
}

func sdkKeysOracleStates(includeNoPrimary bool) []sdkKeysOracleState {
	var states []sdkKeysOracleState

	for members := range uint(64) {
		if bits.OnesCount(members) > 3 {
			continue
		}

		if includeNoPrimary {
			states = append(states, sdkKeysOracleState{members: members, primary: -1})
		}

		for primary := range 6 {
			if members&(1<<primary) != 0 {
				states = append(states, sdkKeysOracleState{members: members, primary: primary})
			}
		}
	}

	return states
}

func sdkKeysOracleReachable(current, desired sdkKeysOracleState) bool {
	queue := []sdkKeysOracleState{current}
	visited := map[sdkKeysOracleState]bool{current: true}

	for len(queue) > 0 {
		state := queue[0]
		queue = queue[1:]

		if state == desired {
			return true
		}

		for symbol := range 6 {
			candidate, legal := sdkKeysOracleStep(state, desired, symbol)
			if legal && !visited[candidate] {
				visited[candidate] = true
				queue = append(queue, candidate)
			}
		}
	}

	return false
}

func sdkKeysOracleStep(state, desired sdkKeysOracleState, symbol int) (sdkKeysOracleState, bool) {
	member := uint(1) << symbol
	present := state.members&member != 0
	wanted := desired.members&member != 0

	switch {
	case !present && wanted && bits.OnesCount(state.members) < 3:
		state.members |= member
		if symbol == desired.primary {
			state.primary = symbol
		}
	case present && !wanted && symbol != state.primary:
		state.members &^= member
	case present && symbol == desired.primary && state.primary != symbol:
		state.primary = symbol
	default:
		return state, false
	}

	return state, true
}

func sdkKeysOracleCollection(state sdkKeysOracleState) []sdkAuthenticationCollectionKey {
	var keys []sdkAuthenticationCollectionKey

	for symbol := range 6 {
		if state.members&(1<<symbol) != 0 {
			keys = append(keys, sdkAuthenticationCollectionKey{
				ID: fmt.Sprintf("id-%d", 6-symbol), RSAPublicKey: fmt.Sprintf("rsa-%d", symbol),
				Description: "shared description", Primary: symbol == state.primary,
			})
		}
	}

	return keys
}

func TestSDKAuthenticationKeysPlannerExhaustiveTransitions(t *testing.T) {
	t.Parallel()

	for _, current := range sdkKeysOracleStates(true) {
		for _, desired := range sdkKeysOracleStates(false) {
			observed, configured := sdkKeysOracleCollection(current), sdkKeysOracleCollection(desired)

			operations, err := planSDKAuthenticationKeys(observed, configured)
			if !sdkKeysOracleReachable(current, desired) {
				require.ErrorIs(t, err, errSDKAuthenticationKeysTemporaryPrimary, "current=%+v desired=%+v", current, desired)
				require.Nil(t, operations)

				continue
			}

			require.NoError(t, err, "current=%+v desired=%+v", current, desired)
			sdkKeysOracleExecute(t, observed, configured, operations)
			require.Equal(t, sdkKeysOracleCollection(current), observed, "planner mutated current input")
			require.Equal(t, sdkKeysOracleCollection(desired), configured, "planner mutated desired input")

			for _, currentOrder := range sdkKeysPermutations(observed) {
				for _, desiredOrder := range sdkKeysPermutations(configured) {
					permuted, permutationErr := planSDKAuthenticationKeys(currentOrder, desiredOrder)
					require.NoError(t, permutationErr)
					require.Equal(t, operations, permuted, "input order changed operations")
				}
			}
		}
	}
}

func sdkKeysPermutations(keys []sdkAuthenticationCollectionKey) [][]sdkAuthenticationCollectionKey {
	if len(keys) == 0 {
		return [][]sdkAuthenticationCollectionKey{nil}
	}

	var permutations [][]sdkAuthenticationCollectionKey

	for i, key := range keys {
		remainder := slices.Delete(slices.Clone(keys), i, i+1)
		for _, suffix := range sdkKeysPermutations(remainder) {
			permutations = append(permutations, append([]sdkAuthenticationCollectionKey{key}, suffix...))
		}
	}

	return permutations
}

func sdkKeysOracleExecute(t *testing.T, current, desired []sdkAuthenticationCollectionKey, operations []sdkAuthenticationKeysOperation) {
	t.Helper()

	state := make(map[string]sdkAuthenticationCollectionKey, len(current))
	wanted := make(map[[2]string]bool, len(desired))
	retained := make(map[string]sdkAuthenticationCollectionKey)

	for _, key := range desired {
		wanted[[2]string{key.RSAPublicKey, key.Description}] = key.Primary
	}

	for _, key := range current {
		state[key.ID] = key
		if _, exists := wanted[[2]string{key.RSAPublicKey, key.Description}]; exists {
			retained[key.ID] = key
		}
	}

	for index, operation := range operations {
		key := operation.Key
		primary, desiredMember := wanted[[2]string{key.RSAPublicKey, key.Description}]

		switch operation.Kind {
		case sdkAuthenticationKeysCreate:
			require.Less(t, len(state), 3, "create exceeded capacity")
			require.True(t, desiredMember)
			require.Equal(t, primary, key.Primary)
			require.Empty(t, key.ID, "create used a configured ID")

			for _, existing := range state {
				require.NotEqual(t, [2]string{existing.RSAPublicKey, existing.Description}, [2]string{key.RSAPublicKey, key.Description}, "recreated an existing member")
			}

			key.ID = fmt.Sprintf("created-%d", index)
			state[key.ID] = key
		case sdkAuthenticationKeysPromote:
			existing, exists := state[key.ID]
			require.True(t, exists, "promoted an absent ID")
			require.True(t, desiredMember && primary && key.Primary, "temporary promotion")
			require.Equal(t, existing.RSAPublicKey, key.RSAPublicKey)
			require.Equal(t, existing.Description, key.Description)
		case sdkAuthenticationKeysDelete:
			existing, exists := state[key.ID]
			require.True(t, exists, "deleted an absent ID")
			require.False(t, desiredMember, "deleted a retained member")
			require.False(t, existing.Primary, "deleted primary")
			require.Equal(t, existing, key)
			delete(state, key.ID)
		default:
			t.Fatalf("unknown operation kind %q", operation.Kind)
		}

		if operation.Kind != sdkAuthenticationKeysDelete && key.Primary {
			for id, existing := range state {
				existing.Primary = id == key.ID
				state[id] = existing
			}
		}
	}

	require.Len(t, state, len(desired))

	for _, key := range state {
		primary, exists := wanted[[2]string{key.RSAPublicKey, key.Description}]
		require.True(t, exists, "unexpected final member")
		require.Equal(t, primary, key.Primary)
	}

	for id, original := range retained {
		key, exists := state[id]
		require.True(t, exists, "lost retained ID")
		require.Equal(t, original.RSAPublicKey, key.RSAPublicKey)
		require.Equal(t, original.Description, key.Description)
	}
}

func TestSDKAuthenticationKeysPlannerOrdering(t *testing.T) {
	t.Parallel()

	current := sdkKeysOracleCollection(sdkKeysOracleState{members: 7, primary: 0})
	desired := sdkKeysOracleCollection(sdkKeysOracleState{members: 10, primary: 3})
	operations, err := planSDKAuthenticationKeys(current, desired)
	require.NoError(t, err)
	require.Len(t, operations, 3)
	assert.Equal(t, sdkAuthenticationKeysDelete, operations[0].Kind)
	assert.Equal(t, "id-4", operations[0].Key.ID)
	assert.Equal(t, sdkAuthenticationKeysCreate, operations[1].Kind)
	assert.True(t, operations[1].Key.Primary)
	assert.Equal(t, sdkAuthenticationKeysDelete, operations[2].Kind)
	assert.Equal(t, "id-6", operations[2].Key.ID)

	emptyOperations, err := planSDKAuthenticationKeys(nil, sdkKeysOracleCollection(sdkKeysOracleState{members: 7, primary: 2}))
	require.NoError(t, err)
	require.Len(t, emptyOperations, 3)
	assert.Equal(t, "rsa-2", emptyOperations[0].Key.RSAPublicKey)
	assert.Equal(t, "rsa-0", emptyOperations[1].Key.RSAPublicKey)
	assert.Equal(t, "rsa-1", emptyOperations[2].Key.RSAPublicKey)
}

func TestSDKAuthenticationKeysPlannerTemporaryPrimary(t *testing.T) {
	t.Parallel()

	current := sdkKeysOracleCollection(sdkKeysOracleState{members: 7, primary: 0})
	desired := sdkKeysOracleCollection(sdkKeysOracleState{members: 14, primary: 3})
	operations, err := planSDKAuthenticationKeys(current, desired)
	require.ErrorIs(t, err, errSDKAuthenticationKeysTemporaryPrimary)
	assert.Nil(t, operations)
	require.ErrorContains(t, err, "all three slots are occupied")
	require.ErrorContains(t, err, "id-6")
	require.ErrorContains(t, err, "Explicitly promote a surviving key in a preceding apply")

	intermediate := sdkKeysOracleCollection(sdkKeysOracleState{members: 7, primary: 1})
	first, err := planSDKAuthenticationKeys(current, intermediate)
	require.NoError(t, err)
	require.Len(t, first, 1)
	assert.Equal(t, sdkAuthenticationKeysPromote, first[0].Kind)
	sdkKeysOracleExecute(t, current, intermediate, first)

	second, err := planSDKAuthenticationKeys(intermediate, desired)
	require.NoError(t, err)
	sdkKeysOracleExecute(t, intermediate, desired, second)
}

func TestSDKAuthenticationKeysPlannerExactMatching(t *testing.T) {
	t.Parallel()

	key := sdkAuthenticationCollectionKey{ID: "original", RSAPublicKey: "opaque PEM\n", Description: "description", Primary: true}

	for name, replacement := range map[string]sdkAuthenticationCollectionKey{
		"different material":    {RSAPublicKey: "opaque PEM", Description: key.Description, Primary: true},
		"different description": {RSAPublicKey: key.RSAPublicKey, Description: "description ", Primary: true},
		"old empty description": {RSAPublicKey: key.RSAPublicKey, Description: key.Description, Primary: true},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			current := key
			if name == "old empty description" {
				current.Description = ""
			}

			operations, err := planSDKAuthenticationKeys([]sdkAuthenticationCollectionKey{current}, []sdkAuthenticationCollectionKey{replacement})
			require.NoError(t, err)
			require.Len(t, operations, 2)
			assert.Equal(t, sdkAuthenticationKeysCreate, operations[0].Kind)
			assert.Equal(t, sdkAuthenticationKeysDelete, operations[1].Kind)
			sdkKeysOracleExecute(t, []sdkAuthenticationCollectionKey{current}, []sdkAuthenticationCollectionKey{replacement}, operations)
		})
	}

	configured := key
	configured.ID = "unrelated configured ID"
	operations, err := planSDKAuthenticationKeys([]sdkAuthenticationCollectionKey{key}, []sdkAuthenticationCollectionKey{configured})
	require.NoError(t, err)
	assert.Empty(t, operations)
}

func TestSDKAuthenticationKeysPlannerMalformedCollections(t *testing.T) {
	t.Parallel()

	valid := sdkAuthenticationCollectionKey{ID: "id", RSAPublicKey: "rsa", Description: "description", Primary: true}
	other := sdkAuthenticationCollectionKey{ID: "other", RSAPublicKey: "other", Description: "description"}
	duplicate := valid
	duplicate.ID, duplicate.Primary = "duplicate", false
	emptyID, emptyMaterial, emptyDescription, noPrimary := valid, valid, valid, valid
	emptyID.ID, emptyMaterial.RSAPublicKey, emptyDescription.Description, noPrimary.Primary = "", "", "", false
	secondPrimary := other
	secondPrimary.Primary = true

	for name, test := range map[string]struct {
		current []sdkAuthenticationCollectionKey
		desired []sdkAuthenticationCollectionKey
	}{
		"empty desired":              {current: []sdkAuthenticationCollectionKey{valid}},
		"too many desired":           {desired: []sdkAuthenticationCollectionKey{valid, other, other, other}},
		"too many current":           {current: []sdkAuthenticationCollectionKey{valid, other, other, other}, desired: []sdkAuthenticationCollectionKey{valid}},
		"empty current ID":           {current: []sdkAuthenticationCollectionKey{emptyID}, desired: []sdkAuthenticationCollectionKey{valid}},
		"duplicate current ID":       {current: []sdkAuthenticationCollectionKey{valid, valid}, desired: []sdkAuthenticationCollectionKey{valid}},
		"duplicate current pair":     {current: []sdkAuthenticationCollectionKey{valid, duplicate}, desired: []sdkAuthenticationCollectionKey{valid}},
		"multiple current primaries": {current: []sdkAuthenticationCollectionKey{valid, secondPrimary}, desired: []sdkAuthenticationCollectionKey{valid}},
		"empty desired material":     {desired: []sdkAuthenticationCollectionKey{emptyMaterial}},
		"empty desired description":  {desired: []sdkAuthenticationCollectionKey{emptyDescription}},
		"no desired primary":         {desired: []sdkAuthenticationCollectionKey{noPrimary}},
		"multiple desired primaries": {desired: []sdkAuthenticationCollectionKey{valid, secondPrimary}},
		"duplicate desired pair":     {desired: []sdkAuthenticationCollectionKey{valid, duplicate}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			operations, err := planSDKAuthenticationKeys(test.current, test.desired)
			require.ErrorIs(t, err, errSDKAuthenticationKeysInvalidCollection)
			assert.Nil(t, operations)
		})
	}

	require.NoError(t, validateSDKAuthenticationKeysCurrent([]sdkAuthenticationCollectionKey{valid, duplicate}), "Read must preserve ambiguous pairs")
}

func TestSDKAuthenticationKeysCollectionsEqual(t *testing.T) {
	t.Parallel()

	keys := sdkKeysOracleCollection(sdkKeysOracleState{members: 7, primary: 0})
	reversed := slices.Clone(keys)
	slices.Reverse(reversed)
	assert.True(t, sdkAuthenticationKeysCollectionsEqual(keys, reversed))
	assert.False(t, sdkAuthenticationKeysCollectionsEqual(keys, keys[:2]))

	for name, mutate := range map[string]func(*sdkAuthenticationCollectionKey){
		"ID":          func(key *sdkAuthenticationCollectionKey) { key.ID = "new ID" },
		"material":    func(key *sdkAuthenticationCollectionKey) { key.RSAPublicKey += "\n" },
		"description": func(key *sdkAuthenticationCollectionKey) { key.Description += " " },
		"primary":     func(key *sdkAuthenticationCollectionKey) { key.Primary = !key.Primary },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			changed := slices.Clone(keys)
			mutate(&changed[0])
			assert.False(t, sdkAuthenticationKeysCollectionsEqual(keys, changed))
		})
	}
}
