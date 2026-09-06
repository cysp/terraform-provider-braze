package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func modifyTemplatePlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse, object string) {
	if req.Plan.Raw.IsNull() {
		resp.Diagnostics.AddWarning(object+" will remain in Braze", "Braze has no delete API for this resource. Applying this plan removes it from Terraform state only. Delete it manually in Braze if needed; creating it again can leave duplicates or encounter a name conflict.")

		return
	}

	// Disclose the lifecycle limitation when creating an object as well.
	// Core may suppress this warning when it replans a forced replacement.
	if req.State.Raw.IsNull() {
		resp.Diagnostics.AddWarning(object+" will remain in Braze", "Braze has no delete API for this resource. Creating it produces a new remote object. If this is a replacement, the previous object remains in Braze; later destruction removes Terraform state only. Import an existing object to resume management without creating another one.")
	}

	var configuredID types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("id"), &configuredID)...)

	if resp.Diagnostics.HasError() || configuredID.IsNull() {
		return
	}

	if !req.State.Raw.IsNull() {
		var priorID types.String
		resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &priorID)...)

		if resp.Diagnostics.HasError() || configuredID.Equal(priorID) {
			return
		}
	}

	resp.Diagnostics.AddAttributeError(path.Root("id"), "Cannot configure generated ID", "Braze generates the ID during creation. Remove id from the resource configuration and use an import block to manage an existing object. An existing configuration may retain its current ID until migrated.")
}
