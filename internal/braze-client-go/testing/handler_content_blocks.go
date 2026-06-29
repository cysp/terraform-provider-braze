package testing

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"sort"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	"github.com/google/uuid"
)

func (h *Handler) ListContentBlocks(_ context.Context, params brazeclient.ListContentBlocksParams) (*brazeclient.ListContentBlocksResponse, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	ids := make([]string, 0, len(h.contentBlocks))
	for id := range h.contentBlocks {
		ids = append(ids, id)
	}

	sort.Strings(ids)

	blocks := make([]brazeclient.ListContentBlocksResponseContentBlock, 0, len(h.contentBlocks))
	for _, id := range ids {
		block := h.contentBlocks[id]
		blocks = append(blocks, brazeclient.ListContentBlocksResponseContentBlock{
			ContentBlockID: block.ContentBlockID,
			Name:           block.Name,
			Tags:           block.Tags,
		})
	}

	return &brazeclient.ListContentBlocksResponse{
		Count:         len(blocks),
		ContentBlocks: paginatedItems(blocks, params.Limit, params.Offset),
	}, nil
}

func (h *Handler) GetContentBlockInfo(_ context.Context, params brazeclient.GetContentBlockInfoParams) (*brazeclient.GetContentBlockInfoResponse, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	block, exists := h.contentBlocks[params.ContentBlockID]
	if !exists {
		return nil, errNotFound
	}

	return block, nil
}

func (h *Handler) CreateContentBlock(_ context.Context, req *brazeclient.CreateContentBlockRequest) (*brazeclient.CreateContentBlockResponse, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if req.Name == "" {
		return nil, newStatusCodeError(http.StatusUnprocessableEntity)
	}

	blockID := uuid.NewString()

	block := &brazeclient.GetContentBlockInfoResponse{
		ContentBlockID: blockID,
		Name:           req.Name,
		Content:        req.Content,
	}

	if req.Description.IsSet() {
		block.Description = req.Description
	}

	if req.Tags.IsSet() {
		if req.Tags.IsNull() {
			block.Tags.SetToNull()
		} else {
			block.Tags.SetTo(slices.Clone(req.Tags.Value))
		}
	}

	h.contentBlocks[blockID] = block

	return &brazeclient.CreateContentBlockResponse{
		ContentBlockID: blockID,
		Message:        "success",
	}, nil
}

func (h *Handler) UpdateContentBlock(_ context.Context, req *brazeclient.UpdateContentBlockRequest) (brazeclient.UpdateContentBlockRes, error) { //nolint:ireturn // ogen requires this interface return type.
	h.mu.Lock()
	defer h.mu.Unlock()

	block, exists := h.contentBlocks[req.ContentBlockID]
	if !exists {
		return nil, fmt.Errorf("content block not found: %w", errNotFound)
	}

	name, nameOk := req.Name.Get()
	if nameOk {
		if name == "" {
			return nil, newStatusCodeError(http.StatusUnprocessableEntity)
		}

		block.Name = name
	}

	if req.Content.IsSet() {
		block.Content = req.Content.Value
	}

	if req.Description.IsSet() {
		if req.Description.IsNull() {
			block.Description.SetToNull()
		} else {
			block.Description = req.Description
		}
	}

	if req.Tags.IsSet() {
		if req.Tags.IsNull() {
			block.Tags.SetToNull()
		} else {
			block.Tags.SetTo(slices.Clone(req.Tags.Value))
		}
	}

	return &brazeclient.UpdateContentBlockCreated{
		ContentBlockID: block.ContentBlockID,
		Message:        "success",
	}, nil
}

func (h *Handler) setContentBlock(contentBlockID, name, content, description string, tags []string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	block := &brazeclient.GetContentBlockInfoResponse{
		ContentBlockID: contentBlockID,
		Name:           name,
		Content:        content,
	}

	if description != "" {
		block.Description = brazeclient.NewOptNilString(description)
	}

	if tags != nil {
		block.Tags.SetTo(slices.Clone(tags))
	} else {
		block.Tags.SetToNull()
	}

	h.contentBlocks[contentBlockID] = block
}
