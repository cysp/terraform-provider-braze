package provider

import (
	"context"
	"iter"
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
	entries iter.Seq2[brazeObjectListEntry[Model], error],
	identityAttribute string,
	listErrorSummary string,
	resourceErrorSummary string,
	yield func(list.ListResult) bool,
) {
	remaining := req.Limit

	for entry, err := range entries {
		if err != nil {
			streamBrazeObjectListError(ctx, req, listErrorSummary, err, yield)

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
			switch {
			case entry.ResourceErr != nil:
				result.Diagnostics.AddError(resourceErrorSummary, detailFromError(entry.ResourceErr))
			case entry.Resource == nil:
				result.Diagnostics.AddError(resourceErrorSummary, "Braze returned no resource data.")
			default:
				result.Diagnostics.Append(result.Resource.Set(ctx, *entry.Resource)...)
			}
		}

		remaining--
		if !yield(result) || remaining <= 0 {
			return
		}
	}
}
