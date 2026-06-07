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

func setIdentityAndState(ctx context.Context, identity stateAttributeValueSettable, state stateValueSettable, contentBlockID string, value any) diag.Diagnostics {
	diags := diag.Diagnostics{}

	diags.Append(identity.SetAttribute(ctx, path.Root("id"), contentBlockID)...)
	diags.Append(state.Set(ctx, value)...)

	return diags
}

func setNamedIdentityAndState(ctx context.Context, identity stateAttributeValueSettable, state stateValueSettable, name string, value any) diag.Diagnostics {
	diags := diag.Diagnostics{}

	diags.Append(identity.SetAttribute(ctx, path.Root("name"), name)...)
	diags.Append(state.Set(ctx, value)...)

	return diags
}

func setCatalogItemIdentityAndState(
	ctx context.Context,
	identity stateAttributeValueSettable,
	state stateValueSettable,
	catalogName string,
	itemID string,
	value any,
) diag.Diagnostics {
	diags := diag.Diagnostics{}

	diags.Append(identity.SetAttribute(ctx, path.Root("catalog_name"), catalogName)...)
	diags.Append(identity.SetAttribute(ctx, path.Root("item_id"), itemID)...)
	diags.Append(state.Set(ctx, value)...)

	return diags
}
