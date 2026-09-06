package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                   = (*brazeSDKAuthenticationKeysResource)(nil)
	_ resource.ResourceWithConfigure      = (*brazeSDKAuthenticationKeysResource)(nil)
	_ resource.ResourceWithIdentity       = (*brazeSDKAuthenticationKeysResource)(nil)
	_ resource.ResourceWithImportState    = (*brazeSDKAuthenticationKeysResource)(nil)
	_ resource.ResourceWithModifyPlan     = (*brazeSDKAuthenticationKeysResource)(nil)
	_ resource.ResourceWithValidateConfig = (*brazeSDKAuthenticationKeysResource)(nil)
)

//nolint:ireturn
func NewBrazeSDKAuthenticationKeysResource() resource.Resource {
	return &brazeSDKAuthenticationKeysResource{}
}

type brazeSDKAuthenticationKeysResource struct {
	providerData brazeProviderData
}

func (r *brazeSDKAuthenticationKeysResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sdk_authentication_keys"
}

func (r *brazeSDKAuthenticationKeysResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = BrazeSDKAuthenticationKeysResourceSchema(ctx)
}

func (r *brazeSDKAuthenticationKeysResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = BrazeSDKAuthenticationKeysResourceIdentitySchema()
}

func (r *brazeSDKAuthenticationKeysResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(setProviderData(req.ProviderData, &r.providerData)...)
}

func (r *brazeSDKAuthenticationKeysResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config brazeSDKAuthenticationKeysModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(validateSDKAuthenticationKeysConfig(ctx, config)...)
}

func (r *brazeSDKAuthenticationKeysResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		resp.Diagnostics.AddWarning("SDK Authentication Keys retained", "Destroy only removes this app's key collection from Terraform state. All SDK Authentication keys remain in Braze.")

		return
	}

	// ignore_changes can retain observations that are not valid desired
	// collections. An unchanged plan requires no reconciliation.
	if req.Plan.Raw.Equal(req.State.Raw) {
		return
	}

	var plan, state brazeSDKAuthenticationKeysModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Replacement creation is planned from configuration; ignored keys in this
	// plan still belong to the old app.
	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

		if resp.Diagnostics.HasError() || !state.AppID.Equal(plan.AppID) {
			return
		}
	}

	resp.Diagnostics.Append(validateSDKAuthenticationKeysConfig(ctx, plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	desired, known, diags := plan.collection(ctx, false)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() || !known {
		return
	}

	var current []sdkAuthenticationCollectionKey

	if !req.State.Raw.IsNull() {
		current, known, diags = state.collection(ctx, true)
		resp.Diagnostics.Append(diags...)

		if resp.Diagnostics.HasError() || !known {
			return
		}
	}

	_, err := planSDKAuthenticationKeys(current, desired)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("keys"), "Cannot reconcile SDK Authentication Keys", detailFromError(err))
	}
}

func (r *brazeSDKAuthenticationKeysResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	appID := types.StringValue(req.ID)
	if req.ID == "" && req.Identity != nil {
		resp.Diagnostics.Append(req.Identity.GetAttribute(ctx, path.Root("app_id"), &appID)...)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	if appID.IsNull() || appID.IsUnknown() || strings.TrimSpace(appID.ValueString()) == "" {
		resp.Diagnostics.AddError("Invalid import identifier", "Import an SDK Authentication key collection with its non-empty app ID, or an identity containing app_id.")

		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("app_id"), appID)...)
	resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("app_id"), appID)...)
}

func (r *brazeSDKAuthenticationKeysResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan brazeSDKAuthenticationKeysModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	desired, diags := sdkAuthenticationKeysApplyConfiguration(ctx, plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	client := r.providerData.sdkAuthenticationKeyCollection

	current, err := client.List(ctx, plan.AppID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read SDK Authentication Keys before creation", detailFromError(err))

		return
	}

	if len(current) != 0 {
		resp.Diagnostics.AddError("SDK Authentication Keys require import", "This app already has SDK Authentication keys. Import the app's collection before applying changes so every removal is visible in the plan. Replacement retains existing keys and cannot recover an occupied collection automatically.")

		return
	}

	result, err := reconcileSDKAuthenticationKeys(ctx, client, plan.AppID.ValueString(), current, desired)
	if result.Attempted || err == nil {
		resp.Diagnostics.Append(setSDKAuthenticationKeysResult(ctx, resp.Identity, &resp.State, plan.AppID.ValueString(), result)...)
	}

	if err != nil {
		resp.Diagnostics.AddError("Failed to create SDK Authentication Keys", detailFromError(err)+" Inspect the remote collection and retained state before retrying. A failed initial create may taint this resource; resolve the failure and deliberately untaint it, or relinquish its state entry and import the app again. Do not use replacement as automatic recovery.")
	}
}

func (r *brazeSDKAuthenticationKeysResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state brazeSDKAuthenticationKeysModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	keys, err := r.providerData.sdkAuthenticationKeyCollection.List(ctx, state.AppID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read SDK Authentication Keys", detailFromError(err))

		return
	}

	resp.Diagnostics.Append(setSDKAuthenticationKeysResult(ctx, resp.Identity, &resp.State, state.AppID.ValueString(), sdkAuthenticationKeysResult{Keys: keys, PrimaryVerified: true})...)
}

func (r *brazeSDKAuthenticationKeysResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state brazeSDKAuthenticationKeysModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	desired, diags := sdkAuthenticationKeysApplyConfiguration(ctx, plan)
	resp.Diagnostics.Append(diags...)
	plannedCurrent, known, diags := state.collection(ctx, true)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	client := r.providerData.sdkAuthenticationKeyCollection

	current, err := client.List(ctx, plan.AppID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read SDK Authentication Keys before update", detailFromError(err))

		return
	}

	if !known || !sdkAuthenticationKeysCollectionsEqual(current, plannedCurrent) {
		resp.Diagnostics.Append(setSDKAuthenticationKeysResult(ctx, resp.Identity, &resp.State, plan.AppID.ValueString(), sdkAuthenticationKeysResult{Keys: current, PrimaryVerified: true})...)
		resp.Diagnostics.AddError("SDK Authentication Keys changed since planning", "The app's key collection no longer matches the planned snapshot. Create a fresh plan before applying changes; no keys were mutated.")

		return
	}

	result, err := reconcileSDKAuthenticationKeys(ctx, client, plan.AppID.ValueString(), current, desired)
	resp.Diagnostics.Append(setSDKAuthenticationKeysResult(ctx, resp.Identity, &resp.State, plan.AppID.ValueString(), result)...)

	if err != nil {
		resp.Diagnostics.AddError("Failed to update SDK Authentication Keys", detailFromError(err))
	}
}

// Deleting the management resource intentionally leaves every remote key intact.
func (r *brazeSDKAuthenticationKeysResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func sdkAuthenticationKeysApplyConfiguration(ctx context.Context, plan brazeSDKAuthenticationKeysModel) ([]sdkAuthenticationCollectionKey, diag.Diagnostics) {
	diags := validateSDKAuthenticationKeysConfig(ctx, plan)
	keys, known, collectionDiags := plan.collection(ctx, false)
	diags.Append(collectionDiags...)

	if !known || plan.AppID.IsUnknown() || plan.AppID.IsNull() {
		diags.AddError("Unknown SDK Authentication Keys configuration", "The app ID and all configured key values must be known before any keys can be mutated.")
	}

	return keys, diags
}

func setSDKAuthenticationKeysResult(ctx context.Context, identity stateAttributeValueSettable, state stateValueSettable, appID string, result sdkAuthenticationKeysResult) diag.Diagnostics {
	model, diags := newBrazeSDKAuthenticationKeysModel(ctx, appID, result.Keys, result.PrimaryVerified)
	if diags.HasError() {
		return diags
	}

	diags.Append(identity.SetAttribute(ctx, path.Root("app_id"), appID)...)
	diags.Append(state.Set(ctx, &model)...)

	return diags
}
