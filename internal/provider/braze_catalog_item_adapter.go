package provider

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const catalogItemListPageSize = 50

type catalogItemClient interface {
	Create(ctx context.Context, plan brazeCatalogItemModel) (brazeCatalogItemModel, error)
	Read(ctx context.Context, catalogName, itemID string) (brazeCatalogItemModel, error)
	Update(ctx context.Context, plan brazeCatalogItemModel) (brazeCatalogItemModel, error)
	Delete(ctx context.Context, catalogName, itemID string) error
	List(ctx context.Context, catalogName string, limit int64) ([]brazeObjectListEntry[brazeCatalogItemModel], error)
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
	response, createErr := c.client.CreateCatalogItem(ctx, &request, params)

	tflog.Info(ctx, "braze_catalog_item.create", map[string]any{"params": params, "response": response, "err": createErr})

	if createErr != nil {
		return brazeCatalogItemModel{}, fmt.Errorf("create catalog item: %w", createErr)
	}

	return c.Read(ctx, plan.CatalogName.ValueString(), plan.ItemID.ValueString())
}

func (c generatedCatalogItemClient) Read(ctx context.Context, catalogName, itemID string) (brazeCatalogItemModel, error) {
	params := brazeclient.GetCatalogItemParams{CatalogName: catalogName, ItemID: itemID}
	response, getErr := c.client.GetCatalogItem(ctx, params)

	tflog.Info(ctx, "braze_catalog_item.read", map[string]any{"params": params, "response": response, "err": getErr})

	if getErr != nil {
		return brazeCatalogItemModel{}, classifyBrazeObjectReadError(getErr)
	}

	if response == nil || len(response.GetItems()) == 0 {
		return brazeCatalogItemModel{}, errBrazeObjectEmptyResponse
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
	response, updateErr := c.client.ReplaceCatalogItem(ctx, &request, params)

	tflog.Info(ctx, "braze_catalog_item.update", map[string]any{"params": params, "response": response, "err": updateErr})

	if updateErr != nil {
		return brazeCatalogItemModel{}, fmt.Errorf("replace catalog item: %w", updateErr)
	}

	return c.Read(ctx, plan.CatalogName.ValueString(), plan.ItemID.ValueString())
}

func (c generatedCatalogItemClient) Delete(ctx context.Context, catalogName, itemID string) error {
	params := brazeclient.DeleteCatalogItemParams{CatalogName: catalogName, ItemID: itemID}
	response, deleteErr := c.client.DeleteCatalogItem(ctx, params)

	tflog.Info(ctx, "braze_catalog_item.delete", map[string]any{"params": params, "response": response, "err": deleteErr})

	if deleteErr != nil {
		return classifyBrazeObjectReadError(deleteErr)
	}

	return nil
}

func (c generatedCatalogItemClient) List(ctx context.Context, catalogName string, limit int64) ([]brazeObjectListEntry[brazeCatalogItemModel], error) {
	params := brazeclient.ListCatalogItemsParams{CatalogName: catalogName}
	items := make([]brazeclient.CatalogItem, 0, catalogItemListPageSize)

	for {
		response, listErr := c.client.ListCatalogItems(ctx, params)

		tflog.Info(ctx, "braze_catalog_item.list", map[string]any{"params": params, "response": response, "err": listErr})

		if listErr != nil {
			return nil, fmt.Errorf("list catalog items: %w", listErr)
		}

		if response == nil {
			return nil, errBrazeObjectEmptyResponse
		}

		pageResponse := response.GetResponse()

		page := pageResponse.GetItems()
		for _, item := range page {
			if int64(len(items)) >= limit {
				break
			}

			items = append(items, item)
		}

		if int64(len(items)) >= limit {
			break
		}

		nextCursor, ok := nextCursorFromLinkHeader(response.GetLink())
		if !ok {
			break
		}

		params.Cursor.SetTo(nextCursor)
	}

	entries := make([]brazeObjectListEntry[brazeCatalogItemModel], 0, len(items))
	for _, item := range items {
		model, err := newBrazeCatalogItemModelFromCatalogItem(catalogName, item)

		entry := brazeObjectListEntry[brazeCatalogItemModel]{
			ID:          catalogName + "/" + item.GetID(),
			DisplayName: item.GetID(),
		}

		if err != nil {
			entry.ResourceErr = err
		} else {
			entry.Resource = &model
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

func nextCursorFromLinkHeader(link brazeclient.OptString) (string, bool) {
	value, ok := link.Get()
	if !ok {
		return "", false
	}

	for value := range strings.SplitSeq(value, ",") {
		linkTarget, linkParams, ok := strings.Cut(value, ";")
		if !ok {
			continue
		}

		isNext := false

		for section := range strings.SplitSeq(linkParams, ";") {
			if strings.TrimSpace(section) == `rel="next"` {
				isNext = true

				break
			}
		}

		if !isNext {
			continue
		}

		rawURL := strings.Trim(strings.TrimSpace(linkTarget), "<>")

		parsed, err := url.Parse(rawURL)
		if err != nil {
			continue
		}

		cursor := parsed.Query().Get("cursor")
		if cursor != "" {
			return cursor, true
		}
	}

	return "", false
}
