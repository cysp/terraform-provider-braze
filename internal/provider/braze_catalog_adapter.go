package provider

import (
	"context"
	"errors"
	"fmt"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var errCatalogNotFound = errors.New("catalog not found")

type catalogClient interface {
	Create(ctx context.Context, plan brazeCatalogModel) (brazeCatalogModel, error)
	Read(ctx context.Context, name string) (brazeCatalogModel, error)
	Delete(ctx context.Context, name string) error
	List(ctx context.Context) ([]brazeObjectListEntry[brazeCatalogModel], error)
}

type generatedCatalogClient struct {
	client *brazeclient.Client
}

func newGeneratedCatalogClient(client *brazeclient.Client) generatedCatalogClient {
	return generatedCatalogClient{client: client}
}

func (c generatedCatalogClient) Create(ctx context.Context, plan brazeCatalogModel) (brazeCatalogModel, error) {
	createRequest, err := plan.ToCreateCatalogRequest(ctx)
	if err != nil {
		return brazeCatalogModel{}, fmt.Errorf("build create catalog request: %w", err)
	}

	createResponse, createErr := c.client.CreateCatalog(ctx, &createRequest)

	tflog.Info(ctx, "braze_catalog.create", map[string]any{
		"request":  createRequest,
		"response": createResponse,
		"err":      createErr,
	})

	if createErr != nil {
		return brazeCatalogModel{}, fmt.Errorf("create catalog: %w", createErr)
	}

	if createResponse == nil || len(createResponse.GetCatalogs()) == 0 {
		return brazeCatalogModel{}, errBrazeObjectEmptyResponse
	}

	return newBrazeCatalogModelFromCatalog(ctx, createResponse.GetCatalogs()[0])
}

func (c generatedCatalogClient) Read(ctx context.Context, name string) (brazeCatalogModel, error) {
	listResponse, listErr := c.client.ListCatalogs(ctx)

	tflog.Info(ctx, "braze_catalog.read", map[string]any{
		"name":     name,
		"response": listResponse,
		"err":      listErr,
	})

	if listErr != nil {
		return brazeCatalogModel{}, fmt.Errorf("list catalogs: %w", listErr)
	}

	if listResponse == nil {
		return brazeCatalogModel{}, errBrazeObjectEmptyResponse
	}

	for _, catalog := range listResponse.GetCatalogs() {
		if catalog.GetName() == name {
			return newBrazeCatalogModelFromCatalog(ctx, catalog)
		}
	}

	return brazeCatalogModel{}, brazeObjectNotFoundError{err: fmt.Errorf("%w: %s", errCatalogNotFound, name)}
}

func (c generatedCatalogClient) Delete(ctx context.Context, name string) error {
	params := brazeclient.DeleteCatalogParams{CatalogName: name}
	deleteResponse, deleteErr := c.client.DeleteCatalog(ctx, params)

	tflog.Info(ctx, "braze_catalog.delete", map[string]any{
		"params":   params,
		"response": deleteResponse,
		"err":      deleteErr,
	})

	if deleteErr != nil {
		return classifyBrazeObjectReadError(deleteErr)
	}

	return nil
}

func (c generatedCatalogClient) List(ctx context.Context) ([]brazeObjectListEntry[brazeCatalogModel], error) {
	listResponse, listErr := c.client.ListCatalogs(ctx)

	tflog.Info(ctx, "braze_catalog.list", map[string]any{
		"response": listResponse,
		"err":      listErr,
	})

	if listErr != nil {
		return nil, fmt.Errorf("list catalogs: %w", listErr)
	}

	if listResponse == nil {
		return nil, errBrazeObjectEmptyResponse
	}

	catalogs := listResponse.GetCatalogs()

	entries := make([]brazeObjectListEntry[brazeCatalogModel], 0, len(catalogs))
	for _, catalog := range catalogs {
		model, err := newBrazeCatalogModelFromCatalog(ctx, catalog)

		entry := brazeObjectListEntry[brazeCatalogModel]{
			ID:          catalog.GetName(),
			DisplayName: catalog.GetName(),
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
