package provider

import (
	"errors"
	"time"
)

const brazeObjectListPageLimit = 100

var errBrazeObjectEmptyResponse = errors.New("empty Braze object response")

type brazeObjectListQuery struct {
	Limit           int64
	ModifiedAfter   *time.Time
	ModifiedBefore  *time.Time
	IncludeResource bool
}

type brazeObjectListEntry[Model any] struct {
	ID          string
	DisplayName string
	Resource    *Model
	ResourceErr error
}

type brazeObjectListItem[Model any] interface {
	ListEntry() brazeObjectListEntry[Model]
}

type brazeObjectNotFoundError struct {
	err error
}

func (e brazeObjectNotFoundError) Error() string {
	return e.err.Error()
}

func (e brazeObjectNotFoundError) Unwrap() error {
	return e.err
}

func isBrazeObjectNotFound(err error) bool {
	var notFound brazeObjectNotFoundError

	return errors.As(err, &notFound)
}

func collectBrazeObjectPages[Item any](query brazeObjectListQuery, fetch func(offset, limit int) ([]Item, error)) ([]Item, error) {
	if query.Limit <= 0 {
		return nil, nil
	}

	offset := 0
	remaining := query.Limit
	items := make([]Item, 0, min(int(query.Limit), brazeObjectListPageLimit))

	for {
		page, err := fetch(offset, brazeObjectListPageLimit)
		if err != nil {
			return nil, err
		}

		for _, item := range page {
			if remaining <= 0 {
				return items, nil
			}

			items = append(items, item)
			remaining--
		}

		if remaining <= 0 || len(page) < brazeObjectListPageLimit {
			return items, nil
		}

		offset += brazeObjectListPageLimit
	}
}

func buildBrazeObjectListEntries[Item brazeObjectListItem[Model], Model any](
	query brazeObjectListQuery,
	items []Item,
	read func(id string) (Model, error),
) []brazeObjectListEntry[Model] {
	entries := make([]brazeObjectListEntry[Model], 0, len(items))
	for _, item := range items {
		entry := item.ListEntry()

		if query.IncludeResource {
			resource, err := read(entry.ID)
			if err != nil {
				entry.ResourceErr = err
			} else {
				entry.Resource = &resource
			}
		}

		entries = append(entries, entry)
	}

	return entries
}

func listBrazeObjectEntries[Item brazeObjectListItem[Model], Model any](
	query brazeObjectListQuery,
	fetch func(offset, limit int) ([]Item, error),
	read func(id string) (Model, error),
) ([]brazeObjectListEntry[Model], error) {
	items, err := collectBrazeObjectPages(query, fetch)
	if err != nil {
		return nil, err
	}

	return buildBrazeObjectListEntries(query, items, read), nil
}

func applyBrazeObjectListQuery(
	query brazeObjectListQuery,
	offset int,
	limit int,
	setLimit func(int),
	setOffset func(int),
	setModifiedAfter func(time.Time),
	setModifiedBefore func(time.Time),
) {
	setLimit(limit)

	if offset > 0 {
		setOffset(offset)
	}

	if query.ModifiedAfter != nil {
		setModifiedAfter(*query.ModifiedAfter)
	}

	if query.ModifiedBefore != nil {
		setModifiedBefore(*query.ModifiedBefore)
	}
}
