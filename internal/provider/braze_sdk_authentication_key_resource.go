package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                   = (*brazeSDKAuthenticationKeyResource)(nil)
	_ resource.ResourceWithConfigure      = (*brazeSDKAuthenticationKeyResource)(nil)
	_ resource.ResourceWithIdentity       = (*brazeSDKAuthenticationKeyResource)(nil)
	_ resource.ResourceWithImportState    = (*brazeSDKAuthenticationKeyResource)(nil)
	_ resource.ResourceWithModifyPlan     = (*brazeSDKAuthenticationKeyResource)(nil)
	_ resource.ResourceWithValidateConfig = (*brazeSDKAuthenticationKeyResource)(nil)
)

//nolint:ireturn
func NewBrazeSDKAuthenticationKeyResource() resource.Resource {
	return &brazeSDKAuthenticationKeyResource{}
}

type brazeSDKAuthenticationKeyResource struct {
	providerData brazeProviderData
}

// Lifecycle rationale is documented in docs/design/sdk-authentication-keys.md.
func (r *brazeSDKAuthenticationKeyResource) Metadata(
	_ context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_sdk_authentication_key"
}

func (r *brazeSDKAuthenticationKeyResource) IdentitySchema(
	_ context.Context,
	_ resource.IdentitySchemaRequest,
	resp *resource.IdentitySchemaResponse,
) {
	resp.IdentitySchema = BrazeSDKAuthenticationKeyResourceIdentitySchema()
}

func (r *brazeSDKAuthenticationKeyResource) Schema(
	ctx context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = BrazeSDKAuthenticationKeyResourceSchema(ctx)
}

func (r *brazeSDKAuthenticationKeyResource) Configure(
	_ context.Context,
	req resource.ConfigureRequest,
	_ *resource.ConfigureResponse,
) {
	SetProviderDataFromResourceConfigureRequest(req, &r.providerData)
}

func (r *brazeSDKAuthenticationKeyResource) ValidateConfig(
	ctx context.Context,
	req resource.ValidateConfigRequest,
	resp *resource.ValidateConfigResponse,
) {
	var config brazeSDKAuthenticationKeyModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !config.Primary.IsNull() && !config.Primary.IsUnknown() && !config.Primary.ValueBool() {
		resp.Diagnostics.AddAttributeError(
			path.Root("primary"),
			"Invalid primary value",
			"Configure `primary = true` to make this key primary, or omit `primary` to leave primary selection unmanaged. Braze does not support directly unsetting the primary key.",
		)
	}

	requiredStrings := []struct {
		name  string
		value types.String
	}{
		{name: "app_id", value: config.AppID},
		{name: "rsa_public_key", value: config.RSAPublicKey},
		{name: "description", value: config.Description},
	}
	for _, attribute := range requiredStrings {
		if !attribute.value.IsNull() && !attribute.value.IsUnknown() && strings.TrimSpace(attribute.value.ValueString()) == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root(attribute.name),
				"Empty required value",
				"The "+attribute.name+" value must not be empty.",
			)
		}
	}
}

func (r *brazeSDKAuthenticationKeyResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	if req.State.Raw.IsNull() {
		return
	}

	var state brazeSDKAuthenticationKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() || state.Primary.IsNull() || state.Primary.IsUnknown() || !state.Primary.ValueBool() {
		return
	}

	if req.Plan.Raw.IsNull() {
		resp.Diagnostics.AddError(
			"Cannot delete currently primary SDK Authentication Key",
			"Braze rejects deletion of whichever SDK Authentication key it currently reports as primary, "+
				"even when the app has other keys. Promote another key and apply that change first. "+
				"After a subsequent refresh reports this key as non-primary, remove it in a later apply. "+
				"To stop managing the key without deleting it from Braze, use a `removed` block with `destroy = false`.",
		)

		return
	}

	var plan brazeSDKAuthenticationKeyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if sdkAuthenticationKeyReplacementPlanned(state, plan) {
		resp.Diagnostics.AddError(
			"Cannot replace currently primary SDK Authentication Key",
			"The provider rejects replacement of a currently primary key because Terraform normally deletes the existing object first, "+
				"which Braze does not permit. `create_before_destroy` compresses creation, promotion, and deletion into one apply "+
				"and cannot represent a controlled JWT issuer-migration interval. Add the replacement at a second stable resource address, "+
				"promote it, migrate JWT issuers, and remove this key in a later apply.",
		)
	}
}

func (r *brazeSDKAuthenticationKeyResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	if req.ID != "" {
		parts := strings.Split(req.ID, "/")
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			resp.Diagnostics.AddError(
				"Invalid import identifier",
				fmt.Sprintf("Expected an import identifier in the form <app_id>/<key_id>. Got %q.", req.ID),
			)

			return
		}

		resp.Diagnostics.Append(setSDKAuthenticationKeyImportIdentityAndState(ctx, resp, parts[0], parts[1])...)

		return
	}

	if req.Identity == nil {
		resp.Diagnostics.AddError(
			"Invalid import identity",
			"Import SDK Authentication keys with an identity import block containing `app_id` and `id`, "+
				"or with an import identifier in the form <app_id>/<key_id>.",
		)

		return
	}

	var (
		appID types.String
		keyID types.String
	)

	resp.Diagnostics.Append(req.Identity.GetAttribute(ctx, path.Root("app_id"), &appID)...)
	resp.Diagnostics.Append(req.Identity.GetAttribute(ctx, path.Root("id"), &keyID)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if strings.TrimSpace(appID.ValueString()) == "" || strings.TrimSpace(keyID.ValueString()) == "" {
		resp.Diagnostics.AddError(
			"Invalid import identity",
			"Import SDK Authentication keys with an identity import block containing non-empty `app_id` and `id`.",
		)

		return
	}

	resp.Diagnostics.Append(setSDKAuthenticationKeyImportIdentityAndState(
		ctx,
		resp,
		appID.ValueString(),
		keyID.ValueString(),
	)...)
}

func (r *brazeSDKAuthenticationKeyResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan brazeSDKAuthenticationKeyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.providerData.sdkAuthenticationKeys.Create(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create SDK Authentication Key", detailFromError(err))

		return
	}

	resp.Diagnostics.Append(setSDKAuthenticationKeyIdentityAndState(
		ctx,
		resp.Identity,
		&resp.State,
		result.Key.AppID.ValueString(),
		result.Key.ID.ValueString(),
		&result.Key,
	)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if result.VerificationError != nil {
		resp.Diagnostics.AddWarning(
			"SDK Authentication Key created but not verified",
			"Braze returned the new SDK Authentication key ID, so the provider preserved it in Terraform state, "+
				"but the subsequent list request could not verify the key. Fix any `sdk_authentication.keys` permission "+
				"or transient API error and refresh the resource. Verification error: "+detailFromError(result.VerificationError),
		)
	}
}

func (r *brazeSDKAuthenticationKeyResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state brazeSDKAuthenticationKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	data, err := r.providerData.sdkAuthenticationKeys.Read(ctx, state.AppID.ValueString(), state.ID.ValueString())
	if err != nil {
		if isBrazeObjectNotFound(err) {
			resp.Diagnostics.AddWarning("SDK Authentication Key not found", detailFromError(err))
			resp.State.RemoveResource(ctx)

			return
		}

		resp.Diagnostics.AddError("Failed to read SDK Authentication Key", detailFromError(err))

		return
	}

	resp.Diagnostics.Append(setSDKAuthenticationKeyIdentityAndState(
		ctx,
		resp.Identity,
		&resp.State,
		data.AppID.ValueString(),
		data.ID.ValueString(),
		&data,
	)...)
}

func (r *brazeSDKAuthenticationKeyResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan brazeSDKAuthenticationKeyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Primary.IsNull() || plan.Primary.IsUnknown() || !plan.Primary.ValueBool() {
		resp.Diagnostics.AddError(
			"Failed to update SDK Authentication Key",
			"Braze only supports updating a key by making it primary. Configure `primary = true`, or replace the key to change another attribute.",
		)

		return
	}

	data, err := r.providerData.sdkAuthenticationKeys.SetPrimary(ctx, plan.AppID.ValueString(), plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to set primary SDK Authentication Key", detailFromError(err))

		return
	}

	resp.Diagnostics.Append(setSDKAuthenticationKeyIdentityAndState(
		ctx,
		resp.Identity,
		&resp.State,
		data.AppID.ValueString(),
		data.ID.ValueString(),
		&data,
	)...)
}

func (r *brazeSDKAuthenticationKeyResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state brazeSDKAuthenticationKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.providerData.sdkAuthenticationKeys.Delete(ctx, state.AppID.ValueString(), state.ID.ValueString())
	if err != nil && !isBrazeObjectNotFound(err) {
		detail := detailFromError(err)
		if errors.Is(err, errSDKAuthenticationPrimaryDelete) ||
			(!state.Primary.IsNull() && !state.Primary.IsUnknown() && state.Primary.ValueBool()) {
			detail += " Braze rejects deletion of whichever key it currently reports as primary; promote another key first."
		}

		resp.Diagnostics.AddError("Failed to delete SDK Authentication Key", detail)
	}
}

func sdkAuthenticationKeyReplacementPlanned(state, plan brazeSDKAuthenticationKeyModel) bool {
	values := []struct {
		state types.String
		plan  types.String
	}{
		{state: state.AppID, plan: plan.AppID},
		{state: state.RSAPublicKey, plan: plan.RSAPublicKey},
		{state: state.Description, plan: plan.Description},
	}

	for _, value := range values {
		if value.plan.IsNull() || value.plan.IsUnknown() || value.state.ValueString() != value.plan.ValueString() {
			return true
		}
	}

	return false
}

func setSDKAuthenticationKeyImportIdentityAndState(
	ctx context.Context,
	resp *resource.ImportStateResponse,
	appID string,
	keyID string,
) diag.Diagnostics {
	diags := diag.Diagnostics{}

	diags.Append(resp.State.SetAttribute(ctx, path.Root("app_id"), appID)...)
	diags.Append(resp.State.SetAttribute(ctx, path.Root("id"), keyID)...)
	diags.Append(resp.Identity.SetAttribute(ctx, path.Root("app_id"), appID)...)
	diags.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), keyID)...)

	return diags
}
