package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

type stateAttributeValueSettable interface {
	SetAttribute(ctx context.Context, path path.Path, value any) diag.Diagnostics
}

type stateValueSettable interface {
	Set(ctx context.Context, value any) diag.Diagnostics
}

func setIdentity(ctx context.Context, identity stateAttributeValueSettable, state stateAttributeValueSettable, contentBlockID string) diag.Diagnostics {
	diags := diag.Diagnostics{}

	diags.Append(identity.SetAttribute(ctx, path.Root("id"), contentBlockID)...)
	diags.Append(state.SetAttribute(ctx, path.Root("id"), contentBlockID)...)

	return diags
}

func setIdentityAndState(ctx context.Context, identity stateAttributeValueSettable, state stateValueSettable, contentBlockID string, value any) diag.Diagnostics {
	diags := diag.Diagnostics{}

	diags.Append(identity.SetAttribute(ctx, path.Root("id"), contentBlockID)...)
	diags.Append(state.Set(ctx, value)...)

	return diags
}
