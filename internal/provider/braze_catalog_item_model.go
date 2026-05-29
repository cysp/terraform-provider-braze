package provider

import (
	"encoding/json"
	"errors"
	"fmt"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	"github.com/go-faster/jx"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type brazeCatalogItemModel struct {
	ID          types.String         `tfsdk:"id"`
	CatalogName types.String         `tfsdk:"catalog_name"`
	ItemID      types.String         `tfsdk:"item_id"`
	DataJSON    jsontypes.Normalized `tfsdk:"data_json"`
}

var (
	errCatalogItemDataJSONIDMismatch  = errors.New("data_json id must match item_id")
	errCatalogItemDataJSONIDNotString = errors.New("data_json id must be a string")
	errCatalogItemDataJSONNotObject   = errors.New("data_json must be a JSON object")
)

func (m brazeCatalogItemModel) ToCatalogItemWrite() (brazeclient.CatalogItemWrite, error) {
	var data map[string]json.RawMessage

	err := json.Unmarshal([]byte(m.DataJSON.ValueString()), &data)
	if err != nil {
		return nil, fmt.Errorf("parse data_json: %w", err)
	}

	if data == nil {
		return nil, errCatalogItemDataJSONNotObject
	}

	itemID := m.ItemID.ValueString()

	if rawID, ok := data["id"]; ok {
		var dataID string

		err := json.Unmarshal(rawID, &dataID)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", errCatalogItemDataJSONIDNotString, err)
		}

		if dataID != itemID {
			return nil, errCatalogItemDataJSONIDMismatch
		}
	}

	delete(data, "id")

	item := make(brazeclient.CatalogItemWrite, len(data))
	for key, value := range data {
		item[key] = jx.Raw(value)
	}

	return item, nil
}

func newBrazeCatalogItemModelFromCatalogItem(catalogName string, item brazeclient.CatalogItem) (brazeCatalogItemModel, error) {
	data := map[string]json.RawMessage{
		"id": mustMarshalRaw(item.GetID()),
	}

	for key, value := range item.GetAdditionalProps() {
		data[key] = json.RawMessage(value)
	}

	raw, err := json.Marshal(data)
	if err != nil {
		return brazeCatalogItemModel{}, fmt.Errorf("marshal data_json: %w", err)
	}

	itemID := item.GetID()
	id := catalogName + "/" + itemID

	return brazeCatalogItemModel{
		ID:          types.StringValue(id),
		CatalogName: types.StringValue(catalogName),
		ItemID:      types.StringValue(itemID),
		DataJSON:    jsontypes.NewNormalizedValue(string(raw)),
	}, nil
}

func mustMarshalRaw(value string) json.RawMessage {
	raw, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}

	return raw
}
