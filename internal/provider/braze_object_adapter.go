package provider

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"time"
)

const brazeObjectListPageLimit = 100

var errBrazeObjectEmptyResponse = errors.New("empty Braze object response")

var errBrazeObjectIdentityMismatch = errors.New("braze returned an unexpected object identity")

func validateBrazeObjectID(expected, actual string) error {
	if actual == "" || actual != expected {
		return fmt.Errorf("%w: requested %q, received %q", errBrazeObjectIdentityMismatch, expected, actual)
	}

	return nil
}

type brazeObjectListQuery struct {
	Limit           int64
	ModifiedAfter   *time.Time
	ModifiedBefore  *time.Time
	IncludeResource bool
}

type brazeObjectListEntry[Model any] struct {
	ID          string
	DisplayName string
	Identity    map[string]string
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

func listBrazeObjectEntries[Item brazeObjectListItem[Model], Model any](
	ctx context.Context,
	query brazeObjectListQuery,
	fetch func(offset, limit int) ([]Item, error),
	read func(id string) (Model, error),
) iter.Seq2[brazeObjectListEntry[Model], error] {
	return func(yield func(brazeObjectListEntry[Model], error) bool) {
		remaining := query.Limit
		for offset := 0; remaining > 0; offset += brazeObjectListPageLimit {
			err := ctx.Err()
			if err != nil {
				yield(brazeObjectListEntry[Model]{}, err)

				return
			}

			page, err := fetch(offset, brazeObjectListPageLimit)
			if err != nil {
				yield(brazeObjectListEntry[Model]{}, err)

				return
			}

			for _, item := range page {
				err := ctx.Err()
				if err != nil {
					yield(brazeObjectListEntry[Model]{}, err)

					return
				}

				entry := item.ListEntry()
				if query.IncludeResource {
					model, readErr := read(entry.ID)
					entry.ResourceErr = readErr
					entry.Resource = &model
				}

				remaining--
				if !yield(entry, nil) || remaining == 0 {
					return
				}
			}

			if len(page) < brazeObjectListPageLimit {
				return
			}
		}
	}
}
