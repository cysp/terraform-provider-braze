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
	ValuesJSON  jsontypes.Normalized `tfsdk:"values_json"`
}

var (
	errCatalogItemValuesJSONIncludesID = errors.New("values_json must not include id")
	errCatalogItemValuesJSONNotObject  = errors.New("values_json must be a JSON object")
)

func (m brazeCatalogItemModel) ToCatalogItemWrite() (brazeclient.CatalogItemWrite, error) {
	var values map[string]json.RawMessage

	err := json.Unmarshal([]byte(m.ValuesJSON.ValueString()), &values)
	if err != nil {
		return nil, fmt.Errorf("parse values_json: %w", err)
	}

	if values == nil {
		return nil, errCatalogItemValuesJSONNotObject
	}

	if _, ok := values["id"]; ok {
		return nil, errCatalogItemValuesJSONIncludesID
	}

	item := make(brazeclient.CatalogItemWrite, len(values))
	for key, value := range values {
		item[key] = jx.Raw(value)
	}

	return item, nil
}

func newBrazeCatalogItemModelFromCatalogItem(catalogName string, item brazeclient.CatalogItem) (brazeCatalogItemModel, error) {
	values := map[string]json.RawMessage{}

	for key, value := range item.GetAdditionalProps() {
		values[key] = json.RawMessage(value)
	}

	raw, err := json.Marshal(values)
	if err != nil {
		return brazeCatalogItemModel{}, fmt.Errorf("marshal values_json: %w", err)
	}

	itemID := item.GetID()
	id := catalogName + "/" + itemID

	return brazeCatalogItemModel{
		ID:          types.StringValue(id),
		CatalogName: types.StringValue(catalogName),
		ItemID:      types.StringValue(itemID),
		ValuesJSON:  jsontypes.NewNormalizedValue(string(raw)),
	}, nil
}
