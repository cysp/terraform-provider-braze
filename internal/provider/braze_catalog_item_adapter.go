package provider

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"net/url"
	"strings"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var errCatalogItemCursor = errors.New("invalid catalog item pagination cursor")

type catalogItemClient interface {
	Create(ctx context.Context, plan brazeCatalogItemModel) (brazeCatalogItemModel, error)
	Read(ctx context.Context, catalogName, itemID string) (brazeCatalogItemModel, error)
	Update(ctx context.Context, plan brazeCatalogItemModel) (brazeCatalogItemModel, error)
	Delete(ctx context.Context, catalogName, itemID string) error
	List(ctx context.Context, catalogName string, limit int64) iter.Seq2[brazeObjectListEntry[brazeCatalogItemModel], error]
}

type generatedCatalogItemClient struct {
	client *brazeclient.Client
}

func newGeneratedCatalogItemClient(client *brazeclient.Client) generatedCatalogItemClient {
	return generatedCatalogItemClient{client: client}
}

func (c generatedCatalogItemClient) Create(ctx context.Context, plan brazeCatalogItemModel) (brazeCatalogItemModel, error) {
	item, err := plan.ToCatalogItemWrite()
	if err != nil {
		return brazeCatalogItemModel{}, err
	}

	request := brazeclient.CreateCatalogItemRequest{Items: []brazeclient.CatalogItemWrite{item}}
	params := brazeclient.CreateCatalogItemParams{CatalogName: plan.CatalogName.ValueString(), ItemID: plan.ItemID.ValueString()}
	_, createErr := c.client.CreateCatalogItem(ctx, &request, params)

	tflog.Debug(ctx, "braze_catalog_item.create", map[string]any{"params": params})

	if createErr != nil {
		return brazeCatalogItemModel{}, fmt.Errorf("create catalog item: %w", createErr)
	}

	data, err := c.Read(ctx, plan.CatalogName.ValueString(), plan.ItemID.ValueString())
	if err != nil {
		return plan, fmt.Errorf("created catalog item %s but could not read it: %w", plan.ID.ValueString(), err)
	}

	return data, nil
}

func (c generatedCatalogItemClient) Read(ctx context.Context, catalogName, itemID string) (brazeCatalogItemModel, error) {
	params := brazeclient.GetCatalogItemParams{CatalogName: catalogName, ItemID: itemID}
	response, getErr := c.client.GetCatalogItem(ctx, params)

	tflog.Debug(ctx, "braze_catalog_item.read", map[string]any{"params": params})

	if getErr != nil {
		return brazeCatalogItemModel{}, classifyBrazeObjectReadError(getErr)
	}

	if response == nil || len(response.GetItems()) != 1 {
		return brazeCatalogItemModel{}, errBrazeObjectEmptyResponse
	}

	err := validateBrazeObjectID(itemID, response.GetItems()[0].GetID())
	if err != nil {
		return brazeCatalogItemModel{}, err
	}

	return newBrazeCatalogItemModelFromCatalogItem(catalogName, response.GetItems()[0])
}

func (c generatedCatalogItemClient) Update(ctx context.Context, plan brazeCatalogItemModel) (brazeCatalogItemModel, error) {
	item, err := plan.ToCatalogItemWrite()
	if err != nil {
		return brazeCatalogItemModel{}, err
	}

	request := brazeclient.ReplaceCatalogItemRequest{Items: []brazeclient.CatalogItemWrite{item}}
	params := brazeclient.ReplaceCatalogItemParams{CatalogName: plan.CatalogName.ValueString(), ItemID: plan.ItemID.ValueString()}
	_, updateErr := c.client.ReplaceCatalogItem(ctx, &request, params)

	tflog.Debug(ctx, "braze_catalog_item.update", map[string]any{"params": params})

	if updateErr != nil {
		return brazeCatalogItemModel{}, fmt.Errorf("replace catalog item: %w", updateErr)
	}

	return c.Read(ctx, plan.CatalogName.ValueString(), plan.ItemID.ValueString())
}

func (c generatedCatalogItemClient) Delete(ctx context.Context, catalogName, itemID string) error {
	params := brazeclient.DeleteCatalogItemParams{CatalogName: catalogName, ItemID: itemID}
	_, deleteErr := c.client.DeleteCatalogItem(ctx, params)

	tflog.Debug(ctx, "braze_catalog_item.delete", map[string]any{"params": params})

	if deleteErr != nil {
		return classifyBrazeObjectReadError(deleteErr)
	}

	return nil
}

//nolint:gocognit // Keep cursor, cancellation, and consumer termination in one pagination loop.
func (c generatedCatalogItemClient) List(ctx context.Context, catalogName string, limit int64) iter.Seq2[brazeObjectListEntry[brazeCatalogItemModel], error] {
	return func(yield func(brazeObjectListEntry[brazeCatalogItemModel], error) bool) {
		params := brazeclient.ListCatalogItemsParams{CatalogName: catalogName}
		seen := map[string]bool{}

		remaining := limit
		for remaining > 0 {
			err := ctx.Err()
			if err != nil {
				yield(brazeObjectListEntry[brazeCatalogItemModel]{}, err)

				return
			}

			response, err := c.client.ListCatalogItems(ctx, params)
			if err != nil {
				yield(brazeObjectListEntry[brazeCatalogItemModel]{}, fmt.Errorf("list catalog items: %w", err))

				return
			}

			if response == nil {
				yield(brazeObjectListEntry[brazeCatalogItemModel]{}, errBrazeObjectEmptyResponse)

				return
			}

			page := response.GetResponse()
			for _, item := range page.GetItems() {
				err := ctx.Err()
				if err != nil {
					yield(brazeObjectListEntry[brazeCatalogItemModel]{}, err)

					return
				}

				model, err := newBrazeCatalogItemModelFromCatalogItem(catalogName, item)

				entry := brazeObjectListEntry[brazeCatalogItemModel]{
					ID: catalogName + "/" + item.GetID(), DisplayName: item.GetID(),
					Identity: map[string]string{"catalog_name": catalogName, "item_id": item.GetID()}, ResourceErr: err,
				}
				entry.Resource = &model

				remaining--
				if !yield(entry, nil) || remaining == 0 {
					return
				}
			}

			next, err := nextCursorFromLinkHeader(response.GetLink())
			if err != nil {
				yield(brazeObjectListEntry[brazeCatalogItemModel]{}, err)

				return
			}

			if next == "" {
				return
			}

			if seen[next] {
				yield(brazeObjectListEntry[brazeCatalogItemModel]{}, fmt.Errorf("%w: repeated cursor", errCatalogItemCursor))

				return
			}

			seen[next] = true
			params.Cursor.SetTo(next)
		}
	}
}

func nextCursorFromLinkHeader(link brazeclient.OptString) (string, error) {
	header, ok := link.Get()
	if !ok || header == "" {
		return "", nil
	}

	for entry := range strings.SplitSeq(header, ",") {
		target, parameters, found := strings.Cut(strings.TrimSpace(entry), ">")
		if !found || !strings.HasPrefix(target, "<") {
			return "", errCatalogItemCursor
		}

		isNext := false

		for section := range strings.SplitSeq(parameters, ";") {
			key, value, _ := strings.Cut(strings.TrimSpace(section), "=")
			if key == "rel" {
				for relation := range strings.FieldsSeq(strings.Trim(value, "\"")) {
					if relation == "next" {
						isNext = true
					}
				}
			}
		}

		if !isNext {
			continue
		}

		parsed, err := url.Parse(strings.TrimPrefix(target, "<"))
		if err != nil {
			return "", errCatalogItemCursor
		}

		cursor := parsed.Query().Get("cursor")
		if cursor == "" {
			return "", errCatalogItemCursor
		}

		return cursor, nil
	}

	return "", nil
}
