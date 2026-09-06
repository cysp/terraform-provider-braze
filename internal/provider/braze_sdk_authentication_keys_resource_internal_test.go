package provider

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errPluralSDKTestFailure = errors.New("injected mutation boundary failure")

type pluralSDKClientStep struct {
	method string
	key    sdkAuthenticationCollectionKey
	keys   []sdkAuthenticationCollectionKey
	id     string
	err    error
}

// This scripted client asserts individual API calls and returns hand-written
// snapshots. It never derives expectations from the executor's state helpers.
type pluralSDKScriptedClient struct {
	t     *testing.T
	steps []pluralSDKClientStep
	calls int
}

func (client *pluralSDKScriptedClient) List(context.Context, string) ([]sdkAuthenticationCollectionKey, error) {
	step := client.next("list")

	return slices.Clone(step.keys), step.err
}

func (client *pluralSDKScriptedClient) Create(_ context.Context, _ string, key sdkAuthenticationCollectionKey) (string, error) {
	step := client.next("create")
	require.Equal(client.t, step.key, key)

	return step.id, step.err
}

func (client *pluralSDKScriptedClient) SetPrimary(_ context.Context, _, keyID string) ([]sdkAuthenticationCollectionKey, error) {
	step := client.next("promote")
	require.Equal(client.t, step.id, keyID)

	return slices.Clone(step.keys), step.err
}

func (client *pluralSDKScriptedClient) Delete(_ context.Context, _, keyID string) ([]sdkAuthenticationCollectionKey, error) {
	step := client.next("delete")
	require.Equal(client.t, step.id, keyID)

	return slices.Clone(step.keys), step.err
}

func (client *pluralSDKScriptedClient) next(method string) pluralSDKClientStep {
	client.t.Helper()
	require.Less(client.t, client.calls, len(client.steps), "unexpected API call %s", method)
	step := client.steps[client.calls]
	client.calls++
	require.Equal(client.t, step.method, method)

	return step
}

func pluralSDKInternalKey(id string, primary bool) sdkAuthenticationCollectionKey {
	return sdkAuthenticationCollectionKey{ID: id, RSAPublicKey: "rsa-" + id, Description: "description-" + id, Primary: primary}
}

func pluralSDKInternalModel(t *testing.T, keys ...sdkAuthenticationCollectionKey) brazeSDKAuthenticationKeysModel {
	t.Helper()

	model, diags := newBrazeSDKAuthenticationKeysModel(t.Context(), "app", keys, true)
	require.False(t, diags.HasError(), "%v", diags)

	return model
}

func pluralSDKInternalMember() brazeSDKAuthenticationKeysMemberModel {
	return brazeSDKAuthenticationKeysMemberModel{
		ID: types.StringUnknown(), RSAPublicKey: types.StringValue("rsa"),
		Description: types.StringValue("description"), Primary: types.BoolValue(true),
	}
}

func pluralSDKInternalMembers(t *testing.T, members ...brazeSDKAuthenticationKeysMemberModel) brazeSDKAuthenticationKeysModel {
	t.Helper()

	keys, diags := types.SetValueFrom(t.Context(), sdkAuthenticationKeysMemberType(), members)
	require.False(t, diags.HasError(), "%v", diags)

	return brazeSDKAuthenticationKeysModel{AppID: types.StringValue("app"), Keys: keys}
}

func pluralSDKInternalState(t *testing.T, model brazeSDKAuthenticationKeysModel) tfsdk.State {
	t.Helper()

	state := tfsdk.State{Schema: BrazeSDKAuthenticationKeysResourceSchema(t.Context())}
	require.False(t, state.Set(t.Context(), model).HasError())

	return state
}

func pluralSDKInternalAssertState(t *testing.T, state tfsdk.State, keys []sdkAuthenticationCollectionKey, primaryVerified bool) {
	t.Helper()
	require.True(t, state.Raw.IsFullyKnown(), "partial state must never contain unknown values")

	var model brazeSDKAuthenticationKeysModel
	require.False(t, state.Get(t.Context(), &model).HasError())
	assert.Equal(t, "app", model.AppID.ValueString())
	require.Len(t, model.Keys.Elements(), len(keys))

	var members []brazeSDKAuthenticationKeysMemberModel
	require.False(t, model.Keys.ElementsAs(t.Context(), &members, false).HasError())

	for _, key := range keys {
		index := slices.IndexFunc(members, func(member brazeSDKAuthenticationKeysMemberModel) bool { return member.ID.ValueString() == key.ID })
		require.NotEqual(t, -1, index, "lost confirmed key ID %s", key.ID)
		member := members[index]
		assert.Equal(t, key.RSAPublicKey, member.RSAPublicKey.ValueString())
		assert.Equal(t, key.Description, member.Description.ValueString())

		if primaryVerified {
			assert.Equal(t, types.BoolValue(key.Primary), member.Primary)
		} else {
			assert.True(t, member.Primary.IsNull(), "unverified role must be null")
		}
	}
}

func TestSDKAuthenticationKeysModelDefersIncompleteValues(t *testing.T) {
	t.Parallel()

	for name, mutate := range map[string]func(*brazeSDKAuthenticationKeysMemberModel){
		"unknown public key":  func(member *brazeSDKAuthenticationKeysMemberModel) { member.RSAPublicKey = types.StringUnknown() },
		"null public key":     func(member *brazeSDKAuthenticationKeysMemberModel) { member.RSAPublicKey = types.StringNull() },
		"unknown description": func(member *brazeSDKAuthenticationKeysMemberModel) { member.Description = types.StringUnknown() },
		"null description":    func(member *brazeSDKAuthenticationKeysMemberModel) { member.Description = types.StringNull() },
		"unknown primary":     func(member *brazeSDKAuthenticationKeysMemberModel) { member.Primary = types.BoolUnknown() },
		"null primary":        func(member *brazeSDKAuthenticationKeysMemberModel) { member.Primary = types.BoolNull() },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			member := pluralSDKInternalMember()
			mutate(&member)
			model := pluralSDKInternalMembers(t, member)
			keys, known, diags := model.collection(t.Context(), false)
			require.False(t, diags.HasError())
			assert.False(t, known)
			assert.Nil(t, keys)
		})
	}

	model := pluralSDKInternalMembers(t, pluralSDKInternalMember())
	keys, known, diags := model.collection(t.Context(), false)
	require.False(t, diags.HasError())
	assert.True(t, known, "computed unknown IDs do not block desired matching")
	require.Len(t, keys, 1)
	assert.Empty(t, keys[0].ID)
	_, known, diags = model.collection(t.Context(), true)
	require.False(t, diags.HasError())
	assert.False(t, known, "unknown observed IDs cannot define a snapshot")

	member := pluralSDKInternalMember()
	member.ID = types.StringNull()
	_, known, diags = pluralSDKInternalMembers(t, member).collection(t.Context(), true)
	require.False(t, diags.HasError())
	assert.False(t, known, "null observed IDs cannot define a snapshot")

	for _, keys := range []types.Set{
		types.SetNull(sdkAuthenticationKeysMemberType()), types.SetUnknown(sdkAuthenticationKeysMemberType()),
		types.SetValueMust(sdkAuthenticationKeysMemberType(), []attr.Value{types.ObjectUnknown(sdkAuthenticationKeysMemberType().AttrTypes)}),
		types.SetValueMust(sdkAuthenticationKeysMemberType(), []attr.Value{types.ObjectNull(sdkAuthenticationKeysMemberType().AttrTypes)}),
	} {
		model.Keys = keys
		_, known, diags = model.collection(t.Context(), false)
		require.False(t, diags.HasError())
		assert.False(t, known)
	}
}

func TestSDKAuthenticationKeysValidationChecksKnownValuesAmidUnknowns(t *testing.T) {
	t.Parallel()

	for name, test := range map[string]struct {
		mutate    func(*brazeSDKAuthenticationKeysMemberModel, *brazeSDKAuthenticationKeysMemberModel)
		wantError bool
	}{
		"unknown primary defers": {mutate: func(first, second *brazeSDKAuthenticationKeysMemberModel) {
			first.Primary = types.BoolValue(false)
			second.Primary = types.BoolUnknown()
		}},
		"known empty public key": {mutate: func(first, second *brazeSDKAuthenticationKeysMemberModel) {
			first.RSAPublicKey = types.StringValue("")
			second.Primary = types.BoolUnknown()
		}, wantError: true},
		"known empty description": {mutate: func(first, second *brazeSDKAuthenticationKeysMemberModel) {
			first.Description = types.StringValue(" ")
			second.Primary = types.BoolUnknown()
		}, wantError: true},
		"known duplicate pair": {mutate: func(first, second *brazeSDKAuthenticationKeysMemberModel) {
			second.RSAPublicKey = first.RSAPublicKey
			second.Description = first.Description
			second.Primary = types.BoolUnknown()
		}, wantError: true},
		"two known primaries with unknown material": {mutate: func(_, second *brazeSDKAuthenticationKeysMemberModel) {
			second.Primary = types.BoolValue(true)
			second.Description = types.StringUnknown()
		}, wantError: true},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			first, second := pluralSDKInternalMember(), pluralSDKInternalMember()
			second.RSAPublicKey = types.StringValue("other rsa")
			second.Primary = types.BoolValue(false)
			test.mutate(&first, &second)
			model := pluralSDKInternalMembers(t, first, second)
			response := resource.ValidateConfigResponse{}
			state := pluralSDKInternalState(t, model)
			(&brazeSDKAuthenticationKeysResource{}).ValidateConfig(t.Context(), resource.ValidateConfigRequest{Config: tfsdk.Config(state)}, &response)
			assert.Equal(t, test.wantError, response.Diagnostics.HasError(), "%v", response.Diagnostics)
		})
	}
}

func TestSDKAuthenticationKeysValidationReportsIndependentErrors(t *testing.T) {
	t.Parallel()

	badPublicKey, badDescription := pluralSDKInternalMember(), pluralSDKInternalMember()
	badPublicKey.RSAPublicKey = types.StringValue(" ")
	badDescription.RSAPublicKey = types.StringValue("other rsa")
	badDescription.Description = types.StringValue(" ")
	badDescription.Primary = types.BoolValue(false)

	memberErrors := []string{"Each key's rsa_public_key value must not be empty.", "Each key's description value must not be empty."}

	for name, test := range map[string]struct {
		appID       string
		members     []brazeSDKAuthenticationKeysMemberModel
		wantDetails []string
	}{
		"invalid app ID preserves primary count": {appID: " ", members: []brazeSDKAuthenticationKeysMemberModel{pluralSDKInternalMember()}, wantDetails: []string{"The app_id value must not be empty."}},
		"member errors forward":                  {appID: "app", members: []brazeSDKAuthenticationKeysMemberModel{badPublicKey, badDescription}, wantDetails: memberErrors},
		"member errors reversed":                 {appID: "app", members: []brazeSDKAuthenticationKeysMemberModel{badDescription, badPublicKey}, wantDetails: memberErrors},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			model := pluralSDKInternalMembers(t, test.members...)
			model.AppID = types.StringValue(test.appID)
			state := pluralSDKInternalState(t, model)
			response := resource.ValidateConfigResponse{}
			(&brazeSDKAuthenticationKeysResource{}).ValidateConfig(t.Context(), resource.ValidateConfigRequest{Config: tfsdk.Config(state)}, &response)

			details := make([]string, 0, len(response.Diagnostics))
			for _, diagnostic := range response.Diagnostics {
				details = append(details, diagnostic.Detail())
			}

			assert.ElementsMatch(t, test.wantDetails, details)
		})
	}
}

func TestSDKAuthenticationKeysModifyPlanPreflight(t *testing.T) {
	t.Parallel()

	current := pluralSDKInternalModel(t, pluralSDKInternalKey("a", true), pluralSDKInternalKey("b", false), pluralSDKInternalKey("c", false))
	desired := pluralSDKInternalModel(t, pluralSDKInternalKey("b", false), pluralSDKInternalKey("c", false), pluralSDKInternalKey("d", true))

	for name, test := range map[string]struct {
		mutate    func(*brazeSDKAuthenticationKeysModel)
		wantError bool
	}{
		"known impossible transition": {mutate: func(*brazeSDKAuthenticationKeysModel) {}, wantError: true},
		"unknown set deferred": {mutate: func(model *brazeSDKAuthenticationKeysModel) {
			model.Keys = types.SetUnknown(sdkAuthenticationKeysMemberType())
		}},
		"app replacement skips old collection": {mutate: func(model *brazeSDKAuthenticationKeysModel) { model.AppID = types.StringValue("replacement-app") }},
		"unknown app deferred":                 {mutate: func(model *brazeSDKAuthenticationKeysModel) { model.AppID = types.StringUnknown() }},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			model := desired
			test.mutate(&model)

			resourceUnderTest := &brazeSDKAuthenticationKeysResource{}
			plan := lifecyclePlan(t, resourceUnderTest, model)
			response := resource.ModifyPlanResponse{Plan: plan}
			resourceUnderTest.ModifyPlan(t.Context(), resource.ModifyPlanRequest{Plan: plan, State: pluralSDKInternalState(t, current)}, &response)
			assert.Equal(t, test.wantError, response.Diagnostics.HasError(), "%v", response.Diagnostics)

			if test.wantError {
				require.Len(t, response.Diagnostics.Errors(), 1)
				assert.Contains(t, response.Diagnostics.Errors()[0].Detail(), "preceding apply")
			}
		})
	}
}

func TestSDKAuthenticationKeysDestroyAmbiguousState(t *testing.T) {
	t.Parallel()

	first, second := pluralSDKInternalKey("a", true), pluralSDKInternalKey("a", false)
	second.ID = "b"
	model := pluralSDKInternalModel(t, first, second)
	state := pluralSDKInternalState(t, model)
	resourceUnderTest := &brazeSDKAuthenticationKeysResource{} // A nil client proves no network operation is attempted.
	plan := tfsdk.Plan{Schema: state.Schema, Raw: tftypes.NewValue(state.Schema.Type().TerraformType(t.Context()), nil)}
	planResponse := resource.ModifyPlanResponse{Plan: plan}
	resourceUnderTest.ModifyPlan(t.Context(), resource.ModifyPlanRequest{Plan: plan, State: state}, &planResponse)
	require.False(t, planResponse.Diagnostics.HasError())
	require.Len(t, planResponse.Diagnostics.Warnings(), 1)
	assert.Contains(t, planResponse.Diagnostics.Warnings()[0].Detail(), "remain in Braze")

	deleteResponse := resource.DeleteResponse{State: state}
	resourceUnderTest.Delete(t.Context(), resource.DeleteRequest{State: state}, &deleteResponse)
	assert.False(t, deleteResponse.Diagnostics.HasError())
}

func TestSDKAuthenticationKeysExecutorFailuresAtEveryBoundary(t *testing.T) {
	t.Parallel()

	aPrimary, aSecondary := pluralSDKInternalKey("a", true), pluralSDKInternalKey("a", false)
	bSecondary, cSecondary := pluralSDKInternalKey("b", false), pluralSDKInternalKey("c", false)
	dPrimary, eSecondary := pluralSDKInternalKey("d", true), pluralSDKInternalKey("e", false)
	dRequest, eRequest := dPrimary, eSecondary
	dRequest.ID, eRequest.ID = "", ""
	current := []sdkAuthenticationCollectionKey{aPrimary, bSecondary, cSecondary}
	desired := []sdkAuthenticationCollectionKey{bSecondary, dPrimary, eSecondary}
	steps := []pluralSDKClientStep{
		{method: "delete", id: "c", keys: []sdkAuthenticationCollectionKey{aPrimary, bSecondary}},
		{method: "create", id: "d", key: dRequest},
		{method: "list", keys: []sdkAuthenticationCollectionKey{aSecondary, bSecondary, dPrimary}},
		{method: "delete", id: "a", keys: []sdkAuthenticationCollectionKey{bSecondary, dPrimary}},
		{method: "create", id: "e", key: eRequest},
		{method: "list", keys: []sdkAuthenticationCollectionKey{bSecondary, dPrimary, eSecondary}},
	}

	for boundary, test := range []struct {
		name            string
		keys            []sdkAuthenticationCollectionKey
		primaryVerified bool
	}{
		{name: "first deletion", keys: current, primaryVerified: true},
		{name: "primary creation", keys: []sdkAuthenticationCollectionKey{aPrimary, bSecondary}},
		{name: "primary creation verification", keys: []sdkAuthenticationCollectionKey{aSecondary, bSecondary, dPrimary}},
		{name: "second deletion", keys: []sdkAuthenticationCollectionKey{aSecondary, bSecondary, dPrimary}, primaryVerified: true},
		{name: "secondary creation", keys: []sdkAuthenticationCollectionKey{bSecondary, dPrimary}, primaryVerified: true},
		{name: "secondary creation verification", keys: []sdkAuthenticationCollectionKey{bSecondary, dPrimary, eSecondary}, primaryVerified: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			failingSteps := slices.Clone(steps[:boundary+1])
			failingSteps[boundary].err = errPluralSDKTestFailure
			failingSteps[boundary].keys = nil
			client := &pluralSDKScriptedClient{t: t, steps: failingSteps}
			result, err := reconcileSDKAuthenticationKeys(t.Context(), client, "app", current, desired)
			require.ErrorIs(t, err, errPluralSDKTestFailure)
			assert.True(t, result.Attempted)
			assert.Equal(t, boundary+1, client.calls, "executor continued after failed boundary")
			assert.ElementsMatch(t, test.keys, result.Keys)
			assert.Equal(t, test.primaryVerified, result.PrimaryVerified)

			model, diags := newBrazeSDKAuthenticationKeysModel(t.Context(), "app", result.Keys, result.PrimaryVerified)
			require.False(t, diags.HasError())
			pluralSDKInternalAssertState(t, pluralSDKInternalState(t, model), test.keys, test.primaryVerified)
		})
	}
}

func TestSDKAuthenticationKeysExecutorPromotionFailures(t *testing.T) {
	t.Parallel()

	current := []sdkAuthenticationCollectionKey{pluralSDKInternalKey("a", true), pluralSDKInternalKey("b", false)}
	desired := []sdkAuthenticationCollectionKey{pluralSDKInternalKey("a", false), pluralSDKInternalKey("b", true)}

	for name, test := range map[string]struct {
		response        []sdkAuthenticationCollectionKey
		err             error
		primaryVerified bool
	}{
		"ambiguous promotion":            {err: errPluralSDKTestFailure},
		"malformed promotion response":   {err: errSDKAuthenticationKeysMultiplePrimary},
		"contradictory observed primary": {response: current, primaryVerified: true},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client := &pluralSDKScriptedClient{t: t, steps: []pluralSDKClientStep{{method: "promote", id: "b", keys: test.response, err: test.err}}}
			result, err := reconcileSDKAuthenticationKeys(t.Context(), client, "app", current, desired)
			require.Error(t, err)
			assert.Equal(t, 1, client.calls)
			assert.Equal(t, test.primaryVerified, result.PrimaryVerified)
			assert.ElementsMatch(t, current, result.Keys)
			model, diags := newBrazeSDKAuthenticationKeysModel(t.Context(), "app", result.Keys, result.PrimaryVerified)
			require.False(t, diags.HasError())
			pluralSDKInternalAssertState(t, pluralSDKInternalState(t, model), current, test.primaryVerified)
		})
	}
}

func TestSDKAuthenticationKeysExecutorCreateVerificationContradictions(t *testing.T) {
	t.Parallel()

	old := pluralSDKInternalKey("old", true)
	created := pluralSDKInternalKey("new", true)
	request := created
	request.ID = ""
	duplicate := old
	duplicate.Primary = false
	unexpected := pluralSDKInternalKey("unexpected", false)

	for name, test := range map[string]struct {
		observed []sdkAuthenticationCollectionKey
		want     []sdkAuthenticationCollectionKey
	}{
		"missing confirmed ID":                              {observed: []sdkAuthenticationCollectionKey{old}, want: []sdkAuthenticationCollectionKey{old, created}},
		"empty observation preserves confirmed ID":          {observed: []sdkAuthenticationCollectionKey{}, want: []sdkAuthenticationCollectionKey{created}},
		"malformed observation preserves provisional state": {observed: []sdkAuthenticationCollectionKey{old, duplicate}, want: []sdkAuthenticationCollectionKey{pluralSDKInternalKey("old", false), created}},
		"new unreviewed key is retained without pruning":    {observed: []sdkAuthenticationCollectionKey{old, unexpected}, want: []sdkAuthenticationCollectionKey{old, unexpected, created}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client := &pluralSDKScriptedClient{t: t, steps: []pluralSDKClientStep{{method: "create", key: request, id: "new"}, {method: "list", keys: test.observed}}}
			result, err := reconcileSDKAuthenticationKeys(t.Context(), client, "app", []sdkAuthenticationCollectionKey{old}, []sdkAuthenticationCollectionKey{created})
			require.Error(t, err)
			assert.Equal(t, 2, client.calls)
			assert.ElementsMatch(t, test.want, result.Keys)
			assert.False(t, result.PrimaryVerified)
		})
	}
}

func TestSDKAuthenticationKeysResourceCreateFailureRetainsKnownState(t *testing.T) {
	t.Parallel()

	created := pluralSDKInternalKey("new", true)
	request := created
	request.ID = ""

	for name, steps := range map[string][]pluralSDKClientStep{
		"confirmed ID then failed verification": {{method: "list", keys: []sdkAuthenticationCollectionKey{}}, {method: "create", key: request, id: "new"}, {method: "list", err: errPluralSDKTestFailure}},
		"lost create response":                  {{method: "list", keys: []sdkAuthenticationCollectionKey{}}, {method: "create", key: request, err: errPluralSDKTestFailure}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client := &pluralSDKScriptedClient{t: t, steps: steps}
			resourceUnderTest := &brazeSDKAuthenticationKeysResource{providerData: brazeProviderData{sdkAuthenticationKeyCollection: client}}
			member := brazeSDKAuthenticationKeysMemberModel{ID: types.StringUnknown(), RSAPublicKey: types.StringValue(request.RSAPublicKey), Description: types.StringValue(request.Description), Primary: types.BoolValue(true)}
			plan := lifecyclePlan(t, resourceUnderTest, pluralSDKInternalMembers(t, member))
			response := resource.CreateResponse{State: tfsdk.State{Schema: plan.Schema}, Identity: lifecycleIdentity(t, resourceUnderTest)}
			resourceUnderTest.Create(t.Context(), resource.CreateRequest{Plan: plan}, &response)
			require.True(t, response.Diagnostics.HasError())
			assert.Equal(t, len(steps), client.calls)
			assert.Contains(t, response.Diagnostics.Errors()[0].Detail(), "taint")

			var want []sdkAuthenticationCollectionKey
			if name == "confirmed ID then failed verification" {
				want = []sdkAuthenticationCollectionKey{created}
			}

			pluralSDKInternalAssertState(t, response.State, want, false)

			var appID types.String
			require.False(t, response.Identity.GetAttribute(t.Context(), path.Root("app_id"), &appID).HasError())
			assert.Equal(t, "app", appID.ValueString())
		})
	}
}

func TestSDKAuthenticationKeysResourceUpdateRejectsConcurrentChanges(t *testing.T) {
	t.Parallel()

	initial := pluralSDKInternalKey("a", true)

	for name, mutate := range map[string]func(*sdkAuthenticationCollectionKey){
		"ID":          func(key *sdkAuthenticationCollectionKey) { key.ID = "changed" },
		"public key":  func(key *sdkAuthenticationCollectionKey) { key.RSAPublicKey += "changed" },
		"description": func(key *sdkAuthenticationCollectionKey) { key.Description += "changed" },
		"primary":     func(key *sdkAuthenticationCollectionKey) { key.Primary = false },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			changed := initial
			mutate(&changed)
			client := &pluralSDKScriptedClient{t: t, steps: []pluralSDKClientStep{{method: "list", keys: []sdkAuthenticationCollectionKey{changed}}}}
			resourceUnderTest := &brazeSDKAuthenticationKeysResource{providerData: brazeProviderData{sdkAuthenticationKeyCollection: client}}
			plan := lifecyclePlan(t, resourceUnderTest, pluralSDKInternalModel(t, initial, pluralSDKInternalKey("b", false)))
			state := pluralSDKInternalState(t, pluralSDKInternalModel(t, initial))
			response := resource.UpdateResponse{State: state, Identity: lifecycleIdentity(t, resourceUnderTest)}
			resourceUnderTest.Update(t.Context(), resource.UpdateRequest{Plan: plan, State: state}, &response)
			require.True(t, response.Diagnostics.HasError())
			assert.Contains(t, response.Diagnostics.Errors()[0].Summary(), "changed since planning")
			assert.Equal(t, 1, client.calls)
			pluralSDKInternalAssertState(t, response.State, []sdkAuthenticationCollectionKey{changed}, true)
		})
	}
}

func TestSDKAuthenticationKeysResourceUpdatePromotionErrorStoresNullPrimary(t *testing.T) {
	t.Parallel()

	current := []sdkAuthenticationCollectionKey{pluralSDKInternalKey("a", true), pluralSDKInternalKey("b", false)}
	desired := []sdkAuthenticationCollectionKey{pluralSDKInternalKey("a", false), pluralSDKInternalKey("b", true)}
	client := &pluralSDKScriptedClient{t: t, steps: []pluralSDKClientStep{{method: "list", keys: current}, {method: "promote", id: "b", err: errPluralSDKTestFailure}}}
	resourceUnderTest := &brazeSDKAuthenticationKeysResource{providerData: brazeProviderData{sdkAuthenticationKeyCollection: client}}
	plan := lifecyclePlan(t, resourceUnderTest, pluralSDKInternalModel(t, desired...))
	state := pluralSDKInternalState(t, pluralSDKInternalModel(t, current...))
	response := resource.UpdateResponse{State: state, Identity: lifecycleIdentity(t, resourceUnderTest)}
	resourceUnderTest.Update(t.Context(), resource.UpdateRequest{Plan: plan, State: state}, &response)
	require.True(t, response.Diagnostics.HasError())
	assert.Equal(t, 2, client.calls)
	pluralSDKInternalAssertState(t, response.State, current, false)
}

func TestSDKAuthenticationKeysModelPreservesAmbiguousObservedMembers(t *testing.T) {
	t.Parallel()

	first := pluralSDKInternalKey("a", true)
	first.Description = ""
	second := first
	second.ID, second.Primary = "b", false
	model := pluralSDKInternalModel(t, first, second)
	require.Len(t, model.Keys.Elements(), 2)
	keys, known, diags := model.collection(t.Context(), true)
	require.False(t, diags.HasError())
	assert.True(t, known)
	assert.ElementsMatch(t, []sdkAuthenticationCollectionKey{first, second}, keys)

	for _, element := range model.Keys.Elements() {
		object, ok := element.(types.Object)
		require.True(t, ok)

		var member brazeSDKAuthenticationKeysMemberModel
		require.False(t, object.As(t.Context(), &member, basetypes.ObjectAsOptions{}).HasError())
		assert.Equal(t, types.StringValue(""), member.Description)
	}
}

func TestSDKAuthenticationKeysResourceReadErrorPreservesState(t *testing.T) {
	t.Parallel()

	keys := []sdkAuthenticationCollectionKey{pluralSDKInternalKey("a", true), pluralSDKInternalKey("b", false)}
	client := &pluralSDKScriptedClient{t: t, steps: []pluralSDKClientStep{{method: "list", err: errPluralSDKTestFailure}}}
	resourceUnderTest := &brazeSDKAuthenticationKeysResource{providerData: brazeProviderData{sdkAuthenticationKeyCollection: client}}
	state := pluralSDKInternalState(t, pluralSDKInternalModel(t, keys...))
	response := resource.ReadResponse{State: state, Identity: lifecycleIdentity(t, resourceUnderTest)}
	resourceUnderTest.Read(t.Context(), resource.ReadRequest{State: state}, &response)
	require.True(t, response.Diagnostics.HasError())
	pluralSDKInternalAssertState(t, response.State, keys, true)
}

func TestSDKAuthenticationKeysExecutorRejectsBeforeAnyMutation(t *testing.T) {
	t.Parallel()

	current := []sdkAuthenticationCollectionKey{pluralSDKInternalKey("a", true), pluralSDKInternalKey("b", false), pluralSDKInternalKey("c", false)}
	desired := []sdkAuthenticationCollectionKey{pluralSDKInternalKey("b", false), pluralSDKInternalKey("c", false), pluralSDKInternalKey("d", true)}
	client := &pluralSDKScriptedClient{t: t}
	result, err := reconcileSDKAuthenticationKeys(t.Context(), client, "app", current, desired)
	require.ErrorIs(t, err, errSDKAuthenticationKeysTemporaryPrimary)
	assert.False(t, result.Attempted)
	assert.True(t, result.PrimaryVerified)
	assert.Equal(t, current, result.Keys)
	assert.Zero(t, client.calls)
}

func TestSDKAuthenticationKeysExecutorContradictoryDeletionStops(t *testing.T) {
	t.Parallel()

	current := []sdkAuthenticationCollectionKey{pluralSDKInternalKey("a", true), pluralSDKInternalKey("b", false), pluralSDKInternalKey("c", false)}
	desired := []sdkAuthenticationCollectionKey{pluralSDKInternalKey("a", true), pluralSDKInternalKey("d", false)}
	client := &pluralSDKScriptedClient{t: t, steps: []pluralSDKClientStep{{method: "delete", id: "b", keys: current}}}
	result, err := reconcileSDKAuthenticationKeys(t.Context(), client, "app", current, desired)
	require.ErrorIs(t, err, errSDKAuthenticationKeysUnexpectedCollection)
	assert.True(t, result.PrimaryVerified)
	assert.Equal(t, current, result.Keys)
	assert.Equal(t, 1, client.calls, "must not create or delete another key after contradictory deletion")
}

func TestSDKAuthenticationKeysResourceUnknownApplyPerformsNoReadsOrWrites(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &brazeSDKAuthenticationKeysResource{} // An incomplete apply must not reach the client.
	member := pluralSDKInternalMember()
	member.Description = types.StringUnknown()
	plan := lifecyclePlan(t, resourceUnderTest, pluralSDKInternalMembers(t, member))
	state := pluralSDKInternalState(t, pluralSDKInternalModel(t, pluralSDKInternalKey("a", true)))
	createResponse := resource.CreateResponse{State: tfsdk.State{Schema: plan.Schema}, Identity: lifecycleIdentity(t, resourceUnderTest)}
	resourceUnderTest.Create(t.Context(), resource.CreateRequest{Plan: plan}, &createResponse)
	assert.True(t, createResponse.Diagnostics.HasError())

	updateResponse := resource.UpdateResponse{State: state, Identity: lifecycleIdentity(t, resourceUnderTest)}
	resourceUnderTest.Update(t.Context(), resource.UpdateRequest{Plan: plan, State: state}, &updateResponse)
	assert.True(t, updateResponse.Diagnostics.HasError())
	assert.True(t, state.Raw.Equal(updateResponse.State.Raw))
}
