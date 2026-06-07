package testing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	"github.com/go-faster/jx"
)

const catalogItemsPageSize = 50

var (
	errCatalogAlreadyExists          = errors.New("catalog already exists")
	errCatalogItemAlreadyExists      = errors.New("catalog item already exists")
	errCatalogItemIDInRequestBody    = errors.New("catalog item request body must not include id")
	errExpectedOneCatalog            = errors.New("expected one catalog")
	errExpectedOneCatalogItem        = errors.New("expected one catalog item")
	errInvalidCatalogItemsPageCursor = errors.New("invalid catalog items page cursor")
)

func (h *Handler) ListCatalogs(context.Context) (*brazeclient.ListCatalogsResponse, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	names := make([]string, 0, len(h.catalogs))
	for name := range h.catalogs {
		names = append(names, name)
	}

	sort.Strings(names)

	catalogs := make([]brazeclient.Catalog, 0, len(names))
	for _, name := range names {
		catalog := h.catalogs[name]
		catalog.NumItems = brazeclient.NewOptInt(len(h.catalogItems[name]))
		catalogs = append(catalogs, catalog)
	}

	return &brazeclient.ListCatalogsResponse{Catalogs: catalogs, Message: "success"}, nil
}

func (s *Server) SetCatalog(name string, description string, fields []brazeclient.CatalogField) {
	s.handler.mu.Lock()
	defer s.handler.mu.Unlock()

	s.handler.catalogs[name] = brazeclient.Catalog{
		Name:        name,
		Description: description,
		Fields:      fields,
		NumItems:    brazeclient.NewOptInt(len(s.handler.catalogItems[name])),
		UpdatedAt:   brazeclient.NewOptDateTime(time.Now().UTC()),
	}

	if _, exists := s.handler.catalogItems[name]; !exists {
		s.handler.catalogItems[name] = map[string]brazeclient.CatalogItem{}
	}
}

func (s *Server) SetCatalogItem(catalogName string, itemID string, fields map[string]json.RawMessage) {
	s.handler.mu.Lock()
	defer s.handler.mu.Unlock()

	if _, exists := s.handler.catalogItems[catalogName]; !exists {
		s.handler.catalogItems[catalogName] = map[string]brazeclient.CatalogItem{}
	}

	additional := make(brazeclient.CatalogItemAdditional, len(fields))
	for name, value := range fields {
		additional[name] = jx.Raw(value)
	}

	s.handler.catalogItems[catalogName][itemID] = brazeclient.CatalogItem{
		ID:              itemID,
		AdditionalProps: additional,
	}
}

func (h *Handler) CreateCatalog(_ context.Context, req *brazeclient.CreateCatalogRequest) (*brazeclient.CreateCatalogResponse, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(req.Catalogs) != 1 {
		return nil, errExpectedOneCatalog
	}

	catalog := req.Catalogs[0]
	if _, exists := h.catalogs[catalog.Name]; exists {
		return nil, fmt.Errorf("%w: %s", errCatalogAlreadyExists, catalog.Name)
	}

	catalog.NumItems = brazeclient.NewOptInt(0)
	catalog.UpdatedAt = brazeclient.NewOptDateTime(time.Now().UTC())
	h.catalogs[catalog.Name] = catalog
	h.catalogItems[catalog.Name] = map[string]brazeclient.CatalogItem{}

	return &brazeclient.CreateCatalogResponse{Catalogs: []brazeclient.Catalog{catalog}, Message: "success"}, nil
}

func (h *Handler) DeleteCatalog(_ context.Context, params brazeclient.DeleteCatalogParams) (*brazeclient.DeleteCatalogResponse, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.catalogs[params.CatalogName]; !exists {
		return nil, errNotFound
	}

	delete(h.catalogs, params.CatalogName)
	delete(h.catalogItems, params.CatalogName)

	return &brazeclient.DeleteCatalogResponse{Message: "success"}, nil
}

func (h *Handler) ListCatalogItems(_ context.Context, params brazeclient.ListCatalogItemsParams) (*brazeclient.ListCatalogItemsResponseHeaders, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	itemsByID, exists := h.catalogItems[params.CatalogName]
	if !exists {
		return nil, errNotFound
	}

	ids := make([]string, 0, len(itemsByID))
	for id := range itemsByID {
		ids = append(ids, id)
	}

	sort.Strings(ids)

	offset := 0

	if cursor, ok := params.Cursor.Get(); ok {
		parsed, err := strconv.Atoi(cursor)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", errInvalidCatalogItemsPageCursor, cursor)
		}

		offset = parsed
	}

	if offset > len(ids) {
		offset = len(ids)
	}

	limit := min(offset+catalogItemsPageSize, len(ids))

	items := make([]brazeclient.CatalogItem, 0, limit-offset)
	for _, id := range ids[offset:limit] {
		items = append(items, itemsByID[id])
	}

	response := brazeclient.ListCatalogItemsResponseHeaders{
		Response: brazeclient.ListCatalogItemsResponse{Items: items, Message: "success"},
	}
	if limit < len(ids) {
		response.Link.SetTo(fmt.Sprintf(`</catalogs/%s/items?cursor=%d>; rel="next"`, params.CatalogName, limit))
	}

	return &response, nil
}

func (h *Handler) GetCatalogItem(_ context.Context, params brazeclient.GetCatalogItemParams) (*brazeclient.GetCatalogItemResponse, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	itemsByID, exists := h.catalogItems[params.CatalogName]
	if !exists {
		return nil, errNotFound
	}

	item, exists := itemsByID[params.ItemID]
	if !exists {
		return nil, errNotFound
	}

	return &brazeclient.GetCatalogItemResponse{Items: []brazeclient.CatalogItem{item}, Message: "success"}, nil
}

func (h *Handler) CreateCatalogItem(_ context.Context, req *brazeclient.CreateCatalogItemRequest, params brazeclient.CreateCatalogItemParams) (*brazeclient.CatalogItemOperationResponse, error) {
	return h.upsertCatalogItem(req.Items, params.CatalogName, params.ItemID, false)
}

func (h *Handler) ReplaceCatalogItem(_ context.Context, req *brazeclient.ReplaceCatalogItemRequest, params brazeclient.ReplaceCatalogItemParams) (*brazeclient.CatalogItemOperationResponse, error) {
	return h.upsertCatalogItem(req.Items, params.CatalogName, params.ItemID, true)
}

func (h *Handler) upsertCatalogItem(items []brazeclient.CatalogItemWrite, catalogName, itemID string, replace bool) (*brazeclient.CatalogItemOperationResponse, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	itemsByID, exists := h.catalogItems[catalogName]
	if !exists {
		return nil, errNotFound
	}

	if len(items) != 1 {
		return nil, errExpectedOneCatalogItem
	}

	writeItem := items[0]
	if _, ok := writeItem["id"]; ok {
		return nil, errCatalogItemIDInRequestBody
	}

	if _, exists := itemsByID[itemID]; exists && !replace {
		return nil, fmt.Errorf("%w: %s", errCatalogItemAlreadyExists, itemID)
	}

	itemsByID[itemID] = brazeclient.CatalogItem{
		ID:              itemID,
		AdditionalProps: brazeclient.CatalogItemAdditional(writeItem),
	}

	return &brazeclient.CatalogItemOperationResponse{Message: "success"}, nil
}

func (h *Handler) DeleteCatalogItem(_ context.Context, params brazeclient.DeleteCatalogItemParams) (*brazeclient.DeleteCatalogItemResponse, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	itemsByID, exists := h.catalogItems[params.CatalogName]
	if !exists {
		return nil, errNotFound
	}

	if _, exists := itemsByID[params.ItemID]; !exists {
		return nil, errNotFound
	}

	delete(itemsByID, params.ItemID)

	return &brazeclient.DeleteCatalogItemResponse{Message: "success"}, nil
}
