package testing

import (
	"context"
	"fmt"
	"net/http"
	"slices"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	"github.com/google/uuid"
)

func (h *Handler) ListContentBlocks(_ context.Context, _ brazeclient.ListContentBlocksParams) (*brazeclient.ListContentBlocksResponse, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	blocks := make([]brazeclient.ListContentBlocksResponseContentBlock, 0, len(h.contentBlocks)+len(h.orphanedBlocks))
	for _, block := range h.contentBlocks {
		blocks = append(blocks, brazeclient.ListContentBlocksResponseContentBlock{
			ContentBlockID: block.ContentBlockID,
			Name:           block.Name,
			Tags:           block.Tags,
		})
	}
	for _, block := range h.orphanedBlocks {
		blocks = append(blocks, block)
	}

	return &brazeclient.ListContentBlocksResponse{
		Count:         len(blocks),
		ContentBlocks: blocks,
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

	if req.Tags != nil {
		block.Tags = slices.Clone(req.Tags)
	}

	h.contentBlocks[blockID] = block

	return &brazeclient.CreateContentBlockResponse{
		ContentBlockID: blockID,
		Message:        "success",
	}, nil
}

func (h *Handler) UpdateContentBlock(_ context.Context, req *brazeclient.UpdateContentBlockRequest) (*brazeclient.UpdateContentBlockResponse, error) {
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
		block.Description = req.Description
	}

	if req.Tags != nil {
		block.Tags = slices.Clone(req.Tags)
	}

	return &brazeclient.UpdateContentBlockResponse{
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
		Tags:           slices.Clone(tags),
	}

	if description != "" {
		block.Description = brazeclient.NewOptString(description)
	}

	h.contentBlocks[contentBlockID] = block
}

func (h *Handler) setOrphanedContentBlock(contentBlockID, name string, tags []string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	block := brazeclient.ListContentBlocksResponseContentBlock{
		ContentBlockID: contentBlockID,
		Name:           name,
		Tags:           slices.Clone(tags),
	}

	h.orphanedBlocks[contentBlockID] = block
}
