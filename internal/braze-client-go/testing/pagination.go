package testing

import brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"

func paginatedItems[T any](items []T, limitOpt, offsetOpt brazeclient.OptInt) []T {
	offset := offsetOpt.Or(0)
	if offset >= len(items) {
		return []T{}
	}

	limit, ok := limitOpt.Get()
	if !ok || limit <= 0 {
		return items[offset:]
	}

	end := min(offset+limit, len(items))

	return items[offset:end]
}
