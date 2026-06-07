package provider

import (
	"context"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

func emptyBrazeObjectListResults(_ func(list.ListResult) bool) {}

func streamBrazeObjectListError(ctx context.Context, req list.ListRequest, summary string, err error, yield func(list.ListResult) bool) {
	result := req.NewListResult(ctx)
	result.Diagnostics.AddError(summary, detailFromError(err))

	yield(result)
}

func streamBrazeObjectListEntries[Model any](
	ctx context.Context,
	req list.ListRequest,
	entries []brazeObjectListEntry[Model],
	identityAttribute string,
	resourceErrorSummary string,
	yield func(list.ListResult) bool,
) {
	for i, entry := range entries {
		if int64(i) >= req.Limit {
			return
		}

		result := req.NewListResult(ctx)
		if len(entry.Identity) == 0 {
			result.Diagnostics.Append(result.Identity.SetAttribute(ctx, path.Root(identityAttribute), entry.ID)...)
		} else {
			identityAttributes := make([]string, 0, len(entry.Identity))
			for attribute := range entry.Identity {
				identityAttributes = append(identityAttributes, attribute)
			}

			sort.Strings(identityAttributes)

			for _, attribute := range identityAttributes {
				result.Diagnostics.Append(result.Identity.SetAttribute(ctx, path.Root(attribute), entry.Identity[attribute])...)
			}
		}

		result.DisplayName = entry.DisplayName

		if req.IncludeResource {
			if entry.ResourceErr != nil {
				result.Diagnostics.AddError(resourceErrorSummary, detailFromError(entry.ResourceErr))
			} else {
				result.Diagnostics.Append(result.Resource.Set(ctx, *entry.Resource)...)
			}
		}

		if !yield(result) {
			return
		}
	}
}
